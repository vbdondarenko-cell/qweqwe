package com.linkup.app.core.session

import android.content.Context
import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import android.util.Base64
import java.security.KeyStore
import javax.crypto.Cipher
import javax.crypto.KeyGenerator
import javax.crypto.SecretKey
import javax.crypto.spec.GCMParameterSpec

class SecureOnboardingStore(context: Context) {
    data class Pending(val verificationToken: String, val telegramDeepLink: String, val expiresAtEpochMillis: Long)

    private val prefs = context.getSharedPreferences("linkup.secure.onboarding", Context.MODE_PRIVATE)

    @Synchronized
    fun save(value: Pending) {
        require(value.verificationToken.isNotBlank())
        require(value.telegramDeepLink.isNotBlank())
        require(value.expiresAtEpochMillis > System.currentTimeMillis())

        val cipher = Cipher.getInstance(TRANSFORMATION)
        cipher.init(Cipher.ENCRYPT_MODE, key())
        val raw = (value.verificationToken + "\n" + value.telegramDeepLink).toByteArray(Charsets.UTF_8)
        val encrypted = cipher.doFinal(raw)
        prefs.edit()
            .putString(KEY_CIPHERTEXT, Base64.encodeToString(encrypted, Base64.NO_WRAP))
            .putString(KEY_IV, Base64.encodeToString(cipher.iv, Base64.NO_WRAP))
            .putLong(KEY_EXPIRES_AT, value.expiresAtEpochMillis)
            .apply()
    }

    @Synchronized
    fun load(): Pending? {
        val expiresAt = prefs.getLong(KEY_EXPIRES_AT, 0L)
        if (expiresAt <= System.currentTimeMillis()) {
            clear()
            return null
        }
        val ciphertext = prefs.getString(KEY_CIPHERTEXT, null) ?: return null
        val iv = prefs.getString(KEY_IV, null) ?: return null
        return runCatching {
            val cipher = Cipher.getInstance(TRANSFORMATION)
            cipher.init(Cipher.DECRYPT_MODE, key(), GCMParameterSpec(128, Base64.decode(iv, Base64.NO_WRAP)))
            val decoded = String(cipher.doFinal(Base64.decode(ciphertext, Base64.NO_WRAP)), Charsets.UTF_8)
            val split = decoded.indexOf('\n')
            require(split > 0 && split < decoded.lastIndex)
            Pending(decoded.substring(0, split), decoded.substring(split + 1), expiresAt)
        }.getOrElse {
            clear()
            null
        }
    }

    @Synchronized
    fun clear() {
        prefs.edit().clear().apply()
    }

    private fun key(): SecretKey {
        val keyStore = KeyStore.getInstance(ANDROID_KEY_STORE).apply { load(null) }
        (keyStore.getKey(KEY_ALIAS, null) as? SecretKey)?.let { return it }
        val generator = KeyGenerator.getInstance(KeyProperties.KEY_ALGORITHM_AES, ANDROID_KEY_STORE)
        generator.init(
            KeyGenParameterSpec.Builder(KEY_ALIAS, KeyProperties.PURPOSE_ENCRYPT or KeyProperties.PURPOSE_DECRYPT)
                .setBlockModes(KeyProperties.BLOCK_MODE_GCM)
                .setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_NONE)
                .setKeySize(256)
                .build(),
        )
        return generator.generateKey()
    }

    private companion object {
        const val ANDROID_KEY_STORE = "AndroidKeyStore"
        const val KEY_ALIAS = "linkup.onboarding.aes.v1"
        const val TRANSFORMATION = "AES/GCM/NoPadding"
        const val KEY_CIPHERTEXT = "ciphertext"
        const val KEY_IV = "iv"
        const val KEY_EXPIRES_AT = "expires_at"
    }
}
