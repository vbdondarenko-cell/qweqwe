plugins {
    id("com.android.application")
    id("org.jetbrains.kotlin.android")
    id("org.jetbrains.kotlin.plugin.compose")
}

fun quotedBuildConfig(value: String): String =
    "\"${value.replace("\\", "\\\\").replace("\"", "\\\"")}\""

val releaseApiBaseUrl = providers.gradleProperty("LINKUP_API_BASE_URL").orElse("").get().trim()
val releasePrivacyUrl = providers.gradleProperty("LINKUP_PRIVACY_URL").orElse("").get().trim()
val releaseTermsUrl = providers.gradleProperty("LINKUP_TERMS_URL").orElse("").get().trim()
val releaseKeystoreFile = providers.gradleProperty("LINKUP_KEYSTORE_FILE").orNull?.trim().orEmpty()
val releaseKeystorePassword = providers.gradleProperty("LINKUP_KEYSTORE_PASSWORD").orNull.orEmpty()
val releaseKeyAlias = providers.gradleProperty("LINKUP_KEY_ALIAS").orNull?.trim().orEmpty()
val releaseKeyPassword = providers.gradleProperty("LINKUP_KEY_PASSWORD").orNull.orEmpty()
val releaseSigningConfigured = listOf(
    releaseKeystoreFile,
    releaseKeystorePassword,
    releaseKeyAlias,
    releaseKeyPassword,
).all { it.isNotBlank() }

android {
    namespace = "com.linkup.app"
    compileSdk = 37

    defaultConfig {
        applicationId = "com.linkup.app"
        minSdk = 26
        targetSdk = 37
        versionCode = 1
        versionName = "1.0.0"
        manifestPlaceholders["usesCleartextTraffic"] = "false"
    }

    signingConfigs {
        if (releaseSigningConfigured) {
            create("release") {
                storeFile = file(releaseKeystoreFile)
                storePassword = releaseKeystorePassword
                keyAlias = releaseKeyAlias
                keyPassword = releaseKeyPassword
                enableV1Signing = true
                enableV2Signing = true
                enableV3Signing = true
                enableV4Signing = true
            }
        }
    }

    buildTypes {
        debug {
            applicationIdSuffix = ".debug"
            versionNameSuffix = "-debug"
            buildConfigField("String", "LINKUP_API_BASE_URL", quotedBuildConfig("http://10.0.2.2:8080"))
            buildConfigField("String", "LINKUP_PRIVACY_URL", quotedBuildConfig(""))
            buildConfigField("String", "LINKUP_TERMS_URL", quotedBuildConfig(""))
            manifestPlaceholders["usesCleartextTraffic"] = "true"
        }
        release {
            isMinifyEnabled = false
            buildConfigField("String", "LINKUP_API_BASE_URL", quotedBuildConfig(releaseApiBaseUrl))
            buildConfigField("String", "LINKUP_PRIVACY_URL", quotedBuildConfig(releasePrivacyUrl))
            buildConfigField("String", "LINKUP_TERMS_URL", quotedBuildConfig(releaseTermsUrl))
            manifestPlaceholders["usesCleartextTraffic"] = "false"
            if (releaseSigningConfigured) {
                signingConfig = signingConfigs.getByName("release")
            }
            proguardFiles(
                getDefaultProguardFile("proguard-android-optimize.txt"),
                "proguard-rules.pro",
            )
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    buildFeatures {
        compose = true
        buildConfig = true
    }

    packaging {
        resources.excludes += "/META-INF/{AL2.0,LGPL2.1}"
        resources.excludes += "DebugProbesKt.bin"
    }
}

kotlin {
    jvmToolchain(17)
}

tasks.matching { it.name == "preReleaseBuild" }.configureEach {
    doFirst {
        check(releaseApiBaseUrl.startsWith("https://")) {
            "Release requires LINKUP_API_BASE_URL with an https:// origin"
        }
        check(releasePrivacyUrl.startsWith("https://")) {
            "Release requires LINKUP_PRIVACY_URL with an https:// URL"
        }
        check(releaseTermsUrl.startsWith("https://")) {
            "Release requires LINKUP_TERMS_URL with an https:// URL"
        }
        check(releaseSigningConfigured) {
            "Release signing requires LINKUP_KEYSTORE_FILE, LINKUP_KEYSTORE_PASSWORD, LINKUP_KEY_ALIAS and LINKUP_KEY_PASSWORD"
        }
        check(file(releaseKeystoreFile).isFile) {
            "LINKUP_KEYSTORE_FILE does not point to a readable keystore file"
        }
    }
}

dependencies {
    val composeBom = platform("androidx.compose:compose-bom:2026.08.00")
    implementation(composeBom)
    androidTestImplementation(composeBom)

    implementation("androidx.activity:activity-compose:1.13.0")
    implementation("androidx.compose.ui:ui")
    implementation("androidx.compose.ui:ui-tooling-preview")
    implementation("androidx.compose.material3:material3")
    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-android:1.11.0")

    testImplementation(kotlin("test"))

    debugImplementation("androidx.compose.ui:ui-tooling")
    debugImplementation("androidx.compose.ui:ui-test-manifest")
}
