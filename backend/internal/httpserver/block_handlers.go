package httpserver

import (
	"errors"
	"net/http"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/blocklist"
)

type blockListResponse struct { Items []blocklist.UserSummary `json:"items"` }

func (s *Server) listBlocks(w http.ResponseWriter,r *http.Request){
	auth,ok:=authFrom(r);if !ok{writeProblem(w,r,http.StatusUnauthorized,"unauthorized","authentication required");return}
	if s.deps.Blocks==nil{writeProblem(w,r,http.StatusServiceUnavailable,"not_ready","block service is unavailable");return}
	items,err:=s.deps.Blocks.List(r.Context(),auth.User.ID);if err!=nil{writeProblem(w,r,http.StatusInternalServerError,"internal_error","request failed");return}
	writeJSON(w,http.StatusOK,blockListResponse{Items:items})
}

func (s *Server) blockUser(w http.ResponseWriter,r *http.Request){
	auth,ok:=authFrom(r);if !ok{writeProblem(w,r,http.StatusUnauthorized,"unauthorized","authentication required");return}
	if s.deps.Blocks==nil{writeProblem(w,r,http.StatusServiceUnavailable,"not_ready","block service is unavailable");return}
	if err:=s.deps.Blocks.Block(r.Context(),auth.User.ID,r.PathValue("userID"));err!=nil{if errors.Is(err,blocklist.ErrInvalidTarget){writeProblem(w,r,http.StatusBadRequest,"invalid_block_target","invalid block target");return};writeProblem(w,r,http.StatusInternalServerError,"internal_error","request failed");return}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) unblockUser(w http.ResponseWriter,r *http.Request){
	auth,ok:=authFrom(r);if !ok{writeProblem(w,r,http.StatusUnauthorized,"unauthorized","authentication required");return}
	if s.deps.Blocks==nil{writeProblem(w,r,http.StatusServiceUnavailable,"not_ready","block service is unavailable");return}
	if err:=s.deps.Blocks.Unblock(r.Context(),auth.User.ID,r.PathValue("userID"));err!=nil{if errors.Is(err,blocklist.ErrInvalidTarget){writeProblem(w,r,http.StatusBadRequest,"invalid_block_target","invalid block target");return};writeProblem(w,r,http.StatusInternalServerError,"internal_error","request failed");return}
	w.WriteHeader(http.StatusNoContent)
}
