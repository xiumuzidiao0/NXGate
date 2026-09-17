package com.nxgate.app

import android.app.Application
import com.nxgate.app.data.ApiClient
import com.nxgate.app.data.ServerStore

class NXGateApplication : Application() {
    lateinit var serverStore: ServerStore
        private set
    lateinit var apiClient: ApiClient
        private set

    override fun onCreate() {
        super.onCreate()
        instance = this
        serverStore = ServerStore(this)
        apiClient = ApiClient()
    }

    companion object {
        lateinit var instance: NXGateApplication
            private set
    }
}
