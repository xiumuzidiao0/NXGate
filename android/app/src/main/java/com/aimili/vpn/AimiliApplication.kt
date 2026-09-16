package com.aimili.vpn

import android.app.Application
import com.aimili.vpn.data.ApiClient
import com.aimili.vpn.data.ServerStore

class AimiliApplication : Application() {
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
        lateinit var instance: AimiliApplication
            private set
    }
}
