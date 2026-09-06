package com.linkup.app

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.Surface
import androidx.compose.ui.Modifier
import com.linkup.app.ui.theme.LinkUpTheme

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            LinkUpTheme {
                // Production feature surfaces are added dependency-first.
                // The approved React/TS design remains the visual contract.
                Surface(modifier = Modifier.fillMaxSize()) {}
            }
        }
    }
}
