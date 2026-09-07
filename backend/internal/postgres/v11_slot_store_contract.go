package postgres

import "github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"

var _ slot.Store = (*V11SlotStore)(nil)
