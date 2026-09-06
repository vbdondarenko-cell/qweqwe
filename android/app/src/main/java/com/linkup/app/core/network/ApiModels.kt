package com.linkup.app.core.network

data class UserProfile(
    val id: String,
    val email: String,
    val username: String,
    val displayName: String,
    val avatarUrl: String?,
    val profileVisibility: String,
    val language: String,
)

data class AuthSession(
    val user: UserProfile,
    val token: String,
    val expiresAtEpochMillis: Long,
)

class ApiException(
    val status: Int,
    val code: String,
    override val message: String,
    val requestId: String? = null,
) : Exception(message)
