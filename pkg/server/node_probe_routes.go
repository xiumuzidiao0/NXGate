package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"aimili-vpngate-go/pkg/tunnel"
)

const (
	maxDeepProbeNodes    = 20
	maxDeepProbeJobsKept = 10
)

type DeepProbeRequest struct {
	NodeIDs []string `json:"node_ids"`
}

type DeepProbeCancelRequest struct {
	JobID string `json:"job_id"`
}

type NodeDeepProbeJob struct {
	JobID      string                        `json:"job_id"`
	Status     string                        `json:"status"` // "running", "completed", "cancelled"
	Total      int                           `json:"total"`
	Completed  int                           `json:"completed"`
	StartedAt  time.Time                     `json:"started_at"`
	FinishedAt *time.Time                    `json:"finished_at,omitempty"`
	Results    []*tunnel.NodeDeepProbeResult `json:"results"`

	cancel context.CancelFunc `json:"-"`
}

func recountCompletedLocked(job *NodeDeepProbeJob) {
	if job == nil {
		return
	}
	done := 0
	for _, r := range job.Results {
		if r != nil && r.Status != "pending" && r.Status != "running" {
			done++
		}
	}
	job.Completed = done
}

func cloneDeepProbeJob(job *NodeDeepProbeJob) *NodeDeepProbeJob {
	if job == nil {
		return nil
	}
	cp := *job
	if job.FinishedAt != nil {
		ft := *job.FinishedAt
		cp.FinishedAt = &ft
	}
	cp.Results = make([]*tunnel.NodeDeepProbeResult, len(job.Results))
	for i, r := range job.Results {
		if r == nil {
			continue
		}
		rc := *r
		if r.Unlock != nil {
			uc := *r.Unlock
			rc.Unlock = &uc
		}
		cp.Results[i] = &rc
	}
	return &cp
}

func (s *Server) handleStartDeepNodeProbe(w http.ResponseWriter, r *http.Request) {
	var req DeepProbeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "请求参数解析失败")
		return
	}

	seen := make(map[string]bool, len(req.NodeIDs))
	uniqueIDs := make([]string, 0, len(req.NodeIDs))
	for _, id := range req.NodeIDs {
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		uniqueIDs = append(uniqueIDs, id)
	}

	if len(uniqueIDs) == 0 {
		s.writeError(w, http.StatusBadRequest, "请至少选择 1 个节点进行实测")
		return
	}
	if len(uniqueIDs) > maxDeepProbeNodes {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("单次最多支持实测 %d 个节点", maxDeepProbeNodes))
		return
	}

	if s.pool == nil || s.tunnelPool == nil {
		s.writeError(w, http.StatusInternalServerError, "节点池或隧道池未初始化")
		return
	}

	jobBudget := time.Duration(len(uniqueIDs))*30*time.Second + 15*time.Second
	if jobBudget > 5*time.Minute {
		jobBudget = 5 * time.Minute
	}
	jobCtx, cancel := context.WithTimeout(context.Background(), jobBudget)

	now := time.Now()
	jobID := fmt.Sprintf("probe-%d", now.UnixNano())
	results := make([]*tunnel.NodeDeepProbeResult, len(uniqueIDs))
	for i, nid := range uniqueIDs {
		node := s.pool.GetNodeByID(nid)
		item := &tunnel.NodeDeepProbeResult{
			NodeID: nid,
			Status: "pending",
		}
		if node != nil {
			item.IP = node.IP
			item.Port = node.Port
			item.Proto = node.Proto
			item.CountryShort = node.CountryShort
			item.SSTPPort = node.SSTPPort
		}
		results[i] = item
	}

	job := &NodeDeepProbeJob{
		JobID:     jobID,
		Status:    "running",
		Total:     len(uniqueIDs),
		Completed: 0,
		StartedAt: now,
		Results:   results,
		cancel:    cancel,
	}

	s.deepProbeMu.Lock()
	if s.deepProbeJobs == nil {
		s.deepProbeJobs = make(map[string]*NodeDeepProbeJob)
	}
	s.deepProbeJobs[jobID] = job
	s.deepProbeOrder = append(s.deepProbeOrder, jobID)
	for len(s.deepProbeOrder) > maxDeepProbeJobsKept {
		evictID := s.deepProbeOrder[0]
		s.deepProbeOrder = s.deepProbeOrder[1:]
		if old := s.deepProbeJobs[evictID]; old != nil && old.cancel != nil {
			old.cancel()
		}
		delete(s.deepProbeJobs, evictID)
	}
	initialSnapshot := cloneDeepProbeJob(job)
	sem := s.deepProbeSem
	if sem == nil {
		sem = make(chan struct{}, 3)
		s.deepProbeSem = sem
	}
	s.deepProbeMu.Unlock()

	go s.runDeepProbeJob(jobCtx, job, uniqueIDs, sem)

	s.writeJSON(w, http.StatusOK, initialSnapshot)
}

func (s *Server) runDeepProbeJob(jobCtx context.Context, job *NodeDeepProbeJob, nodeIDs []string, sem chan struct{}) {
	defer func() {
		if job.cancel != nil {
			job.cancel()
		}
	}()

	var wg sync.WaitGroup
	for idx, nid := range nodeIDs {
		wg.Add(1)
		go func(i int, nodeID string) {
			defer wg.Done()

			select {
			case <-jobCtx.Done():
				s.deepProbeMu.Lock()
				if r := job.Results[i]; r != nil && (r.Status == "pending" || r.Status == "running") {
					r.Status = "cancelled"
					r.Error = "检测已取消"
					now := time.Now()
					r.CheckedAt = now
					recountCompletedLocked(job)
				}
				s.deepProbeMu.Unlock()
				return
			case sem <- struct{}{}:
			}
			defer func() { <-sem }()

			if jobCtx.Err() != nil {
				s.deepProbeMu.Lock()
				if r := job.Results[i]; r != nil && (r.Status == "pending" || r.Status == "running") {
					r.Status = "cancelled"
					r.Error = "检测已取消"
					r.CheckedAt = time.Now()
					recountCompletedLocked(job)
				}
				s.deepProbeMu.Unlock()
				return
			}

			s.deepProbeMu.Lock()
			if job.Results[i] != nil {
				job.Results[i].Status = "running"
			}
			s.deepProbeMu.Unlock()

			node := s.pool.GetNodeByID(nodeID)
			var probeRes *tunnel.NodeDeepProbeResult
			if node == nil {
				probeRes = &tunnel.NodeDeepProbeResult{
					NodeID:    nodeID,
					Status:    "failed",
					Error:     "节点不存在或已从节点库移除",
					CheckedAt: time.Now(),
				}
			} else {
				nodeCtx, nodeCancel := context.WithTimeout(jobCtx, 30*time.Second)
				probeRes = s.tunnelPool.ProbeNodeDeep(nodeCtx, node)
				nodeCancel()
			}

			s.deepProbeMu.Lock()
			job.Results[i] = probeRes
			recountCompletedLocked(job)
			s.deepProbeMu.Unlock()
		}(idx, nid)
	}

	wg.Wait()

	finishedAt := time.Now()
	s.deepProbeMu.Lock()
	if job.Status == "running" {
		if jobCtx.Err() != nil {
			job.Status = "cancelled"
		} else {
			job.Status = "completed"
		}
	}
	job.FinishedAt = &finishedAt
	s.deepProbeMu.Unlock()
}

func (s *Server) handleGetDeepNodeProbe(w http.ResponseWriter, r *http.Request) {
	jobID := r.URL.Query().Get("job_id")

	s.deepProbeMu.Lock()
	defer s.deepProbeMu.Unlock()

	if jobID == "" && len(s.deepProbeOrder) > 0 {
		jobID = s.deepProbeOrder[len(s.deepProbeOrder)-1]
	}
	job := s.deepProbeJobs[jobID]
	if job == nil {
		s.writeError(w, http.StatusNotFound, "未找到对应的节点检测任务")
		return
	}

	s.writeJSON(w, http.StatusOK, cloneDeepProbeJob(job))
}

func (s *Server) handleCancelDeepNodeProbe(w http.ResponseWriter, r *http.Request) {
	var req DeepProbeCancelRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	s.deepProbeMu.Lock()
	jobID := req.JobID
	if jobID == "" && len(s.deepProbeOrder) > 0 {
		jobID = s.deepProbeOrder[len(s.deepProbeOrder)-1]
	}
	job := s.deepProbeJobs[jobID]
	if job == nil {
		s.deepProbeMu.Unlock()
		s.writeError(w, http.StatusNotFound, "未找到对应的节点检测任务")
		return
	}
	if job.Status == "running" {
		job.Status = "cancelled"
		if job.cancel != nil {
			job.cancel()
		}
		now := time.Now()
		for _, res := range job.Results {
			if res != nil && (res.Status == "pending" || res.Status == "running") {
				res.Status = "cancelled"
				res.Error = "检测已取消"
				res.CheckedAt = now
			}
		}
		recountCompletedLocked(job)
	}
	snap := cloneDeepProbeJob(job)
	s.deepProbeMu.Unlock()

	s.writeJSON(w, http.StatusOK, snap)
}
