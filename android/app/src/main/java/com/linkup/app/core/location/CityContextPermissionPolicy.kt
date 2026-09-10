package com.linkup.app.core.location

import com.linkup.app.core.network.CityPermissionClass

internal enum class CityContextPermissionAction {
    UNAVAILABLE,
    REQUEST_PERMISSION,
    RESOLVE_FROM_DEVICE,
}

internal fun cityContextPermissionAction(
    capabilityEnabled: Boolean,
    permissionClass: CityPermissionClass?,
): CityContextPermissionAction = when {
    !capabilityEnabled -> CityContextPermissionAction.UNAVAILABLE
    permissionClass == null -> CityContextPermissionAction.REQUEST_PERMISSION
    else -> CityContextPermissionAction.RESOLVE_FROM_DEVICE
}

internal fun cityLocationPermissionGranted(
    fineGranted: Boolean,
    coarseGranted: Boolean,
    observedPermissionClass: CityPermissionClass?,
): Boolean = fineGranted || coarseGranted || observedPermissionClass != null
