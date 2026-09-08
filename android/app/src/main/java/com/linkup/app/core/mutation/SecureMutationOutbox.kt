package com.linkup.app.core.mutation

import android.content.Context
import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import android.util.Base64
import java.security.KeyStore
import javax.crypto.Cipher
import javax.crypto.KeyGenerator
import javax.crypto.SecretKey
import javax.crypto.spec.GCMParameterSpec
import org.json.JSONArray
import org.json.JSONObject

class SecureMutationOutbox(context: Context) : MutationOutbox {
    private val prefs = context.applicationContext.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)

    @Synchronized
    override fun enqueue(command: DurableMutationCommand) {
        val items = readCommands().toMutableList()
        if (items.any { it.idempotencyKey == command.idempotencyKey }) return
        require(items.size < DURABLE_MUTATION_MAX_COMMANDS) { "mutation outbox is full" }
        items += command
        writeCommands(items)
    }

    @Synchronized
    override fun markAttempt(idempotencyKey: String, nowEpochMillis: Long): DurableMutationCommand? {
        val items = readCommands().toMutableList()
        val index = items.indexOfFirst { it.idempotencyKey == idempotencyKey }
        if (index < 0) return null
        val updated = items[index].markAttempt(nowEpochMillis)
        if (updated != items[index]) {
            items[index] = updated
            writeCommands(items)
        }
        return updated
    }

    @Synchronized
    override fun remove(idempotencyKey: String) {
        val items = readCommands()
        val retained = items.filterNot { it.idempotencyKey == idempotencyKey }
        if (retained.size != items.size) writeCommands(retained)
    }

    @Synchronized
    override fun pendingReplayable(ownerFingerprint: String, nowEpochMillis: Long): List<DurableMutationCommand> =
        readCommands()
            .asSequence()
            .filter { it.ownerFingerprint == ownerFingerprint && it.canAutoReplay(nowEpochMillis) }
            .sortedBy { it.createdAtEpochMillis }
            .toList()

    @Synchronized
    override fun unsafeAmbiguousCount(ownerFingerprint: String, nowEpochMillis: Long): Int =
        readCommands().count {
            it.ownerFingerprint == ownerFingerprint && it.firstAttemptAtEpochMillis != null && !it.canAutoReplay(nowEpochMillis)
        }

    @Synchronized
    override fun clearOwner(ownerFingerprint: String) {
        val items = readCommands()
        val retained = items.filterNot { it.ownerFingerprint == ownerFingerprint }
        if (retained.size != items.size) writeCommands(retained)
    }

    @Synchronized
    override fun clearAll() {
        prefs.edit().clear().apply()
    }

    private fun readCommands(): List<DurableMutationCommand> {
        val ciphertext = prefs.getString(KEY_CIPHERTEXT, null) ?: return emptyList()
        val iv = prefs.getString(KEY_IV, null) ?: return emptyList()
        return runCatching {
            val cipher = Cipher.getInstance(TRANSFORMATION)
            cipher.init(
                Cipher.DECRYPT_MODE,
                key(),
                GCMParameterSpec(128, Base64.decode(iv, Base64.NO_WRAP)),
            )
            val raw = cipher.doFinal(Base64.decode(ciphertext, Base64.NO_WRAP))
            decodeCommands(String(raw, Charsets.UTF_8))
        }.getOrElse {
            prefs.edit().clear().apply()
            emptyList()
        }
    }

    private fun writeCommands(commands: List<DurableMutationCommand>) {
        if (commands.isEmpty()) {
            prefs.edit().clear().apply()
            return
        }
        require(commands.size <= DURABLE_MUTATION_MAX_COMMANDS)
        val plaintext = encodeCommands(commands).toByteArray(Charsets.UTF_8)
        val cipher = Cipher.getInstance(TRANSFORMATION)
        cipher.init(Cipher.ENCRYPT_MODE, key())
        val encrypted = cipher.doFinal(plaintext)
        prefs.edit()
            .putString(KEY_CIPHERTEXT, Base64.encodeToString(encrypted, Base64.NO_WRAP))
            .putString(KEY_IV, Base64.encodeToString(cipher.iv, Base64.NO_WRAP))
            .apply()
    }

    private fun encodeCommands(commands: List<DurableMutationCommand>): String {
        val array = JSONArray()
        commands.forEach { command ->
            array.put(JSONObject()
                .put("idempotencyKey", command.idempotencyKey)
                .put("ownerFingerprint", command.ownerFingerprint)
                .put("method", command.method)
                .put("path", command.path)
                .put("bodyJson", command.bodyJson ?: JSONObject.NULL)
                .put("responseKind", command.responseKind.name)
                .put("createdAtEpochMillis", command.createdAtEpochMillis)
                .put("firstAttemptAtEpochMillis", command.firstAttemptAtEpochMillis ?: JSONObject.NULL))
        }
        return array.toString()
    }

    private fun decodeCommands(raw: String): List<DurableMutationCommand> {
        val array = JSONArray(raw)
        require(array.length() <= DURABLE_MUTATION_MAX_COMMANDS)
        return buildList(array.length()) {
            for (index in 0 until array.length()) {
                val item = array.getJSONObject(index)
                add(DurableMutationCommand(
                    idempotencyKey = item.getString("idempotencyKey"),
                    ownerFingerprint = item.getString("ownerFingerprint"),
                    method = item.getString("method"),
                    path = item.getString("path"),
                    bodyJson = if (item.isNull("bodyJson")) null else item.getString("bodyJson"),
                    responseKind = DurableResponseKind.valueOf(item.getString("responseKind")),
                    createdAtEpochMillis = item.getLong("createdAtEpochMillis"),
                    firstAttemptAtEpochMillis = if (item.isNull("firstAttemptAtEpochMillis")) null else item.getLong("firstAttemptAtEpochMillis"),
                ))
            }
        }
    }

    private fun key(): SecretKey {
        val keyStore = KeyStore.getInstance(ANDROID_KEY_STORE).apply { load(null) }
        (keyStore.getKey(KEY_ALIAS, null) as? SecretKey)?.let { return it }
        val generator = KeyGenerator.getInstance(KeyProperties.KEY_ALGORITHM_AES, ANDROID_KEY_STORE)
        generator.init(
            KeyGenParameterSpec.Builder(
                KEY_ALIAS,
                KeyProperties.PURPOSE_ENCRYPT or KeyProperties.PURPOSE_DECRYPT,
            )
                .setBlockModes(KeyProperties.BLOCK_MODE_GCM)
                .setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_NONE)
                .setKeySize(256)
                .build(),
        )
        return generator.generateKey()
    }

    private companion object {
        const val PREFS_NAME = "linkup.secure.mutation.outbox.v1"
        const val ANDROID_KEY_STORE = "AndroidKeyStore"
        const val KEY_ALIAS = "linkup.mutation.aes.v1"
        const val TRANSFORMATION = "AES/GCM/NoPadding"
        const val KEY_CIPHERTEXT = "ciphertext"
        const val KEY_IV = "iv"
    }
}
