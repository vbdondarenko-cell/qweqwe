package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/blocklist"
)

type BlockStore struct{ pool *pgxpool.Pool }
func NewBlockStore(pool *pgxpool.Pool)*BlockStore{return &BlockStore{pool:pool}}

func (s *BlockStore) Block(ctx context.Context,blockerID,blockedID string,now time.Time)error{
	var exists bool
	if err:=s.pool.QueryRow(ctx,`SELECT EXISTS(SELECT 1 FROM app_users WHERE id=$1)`,blockedID).Scan(&exists);err!=nil{return err}
	if !exists{return blocklist.ErrInvalidTarget}
	_,err:=s.pool.Exec(ctx,`INSERT INTO user_blocks (blocker_id,blocked_id,created_at) VALUES ($1,$2,$3) ON CONFLICT (blocker_id,blocked_id) DO NOTHING`,blockerID,blockedID,now)
	return err
}

func (s *BlockStore) Unblock(ctx context.Context,blockerID,blockedID string)error{
	_,err:=s.pool.Exec(ctx,`DELETE FROM user_blocks WHERE blocker_id=$1 AND blocked_id=$2`,blockerID,blockedID)
	return err
}

func (s *BlockStore) List(ctx context.Context,blockerID string,limit int)([]blocklist.UserSummary,error){
	rows,err:=s.pool.Query(ctx,`SELECT u.id,u.username,u.display_name,u.avatar_url FROM user_blocks b JOIN app_users u ON u.id=b.blocked_id WHERE b.blocker_id=$1 ORDER BY b.created_at DESC,u.id LIMIT $2`,blockerID,limit)
	if err!=nil{return nil,err}
	defer rows.Close()
	out:=make([]blocklist.UserSummary,0)
	for rows.Next(){var item blocklist.UserSummary;if err:=rows.Scan(&item.ID,&item.Username,&item.DisplayName,&item.AvatarURL);err!=nil{return nil,err};out=append(out,item)}
	if err:=rows.Err();err!=nil{return nil,err}
	return out,nil
}
