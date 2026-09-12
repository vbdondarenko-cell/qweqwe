package httpserver

import (
	"errors"
	"net/http"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/account"
)

type registerRequest struct { Email string `json:"email"`; Username string `json:"username"`; DisplayName string `json:"displayName"`; Password string `json:"password"`; Language string `json:"language"`; DeviceLabel string `json:"deviceLabel"` }
type loginRequest struct { Identifier string `json:"identifier"`; Password string `json:"password"`; DeviceLabel string `json:"deviceLabel"` }
type patchMeRequest struct { DisplayName *string `json:"displayName"`; AvatarURL *string `json:"avatarUrl"`; ProfileVisibility *string `json:"profileVisibility"`; Language *string `json:"language"`; Interests *[]string `json:"interests"` }

func (s *Server) register(w http.ResponseWriter,r *http.Request){
	if s.deps.Accounts==nil { writeProblem(w,r,http.StatusServiceUnavailable,"not_ready","account service is unavailable"); return }
	var in registerRequest; if err:=decodeJSON(w,r,&in); err!=nil { writeProblem(w,r,http.StatusBadRequest,"invalid_request","invalid JSON body"); return }
	out,err:=s.deps.Accounts.Register(r.Context(),account.Registration{Email:in.Email,Username:in.Username,DisplayName:in.DisplayName,Password:in.Password,Language:in.Language,DeviceLabel:in.DeviceLabel})
	if err!=nil { s.writeAccountError(w,r,err,false); return }
	writeJSON(w,http.StatusCreated,out)
}

func (s *Server) login(w http.ResponseWriter,r *http.Request){
	if s.deps.Accounts==nil { writeProblem(w,r,http.StatusServiceUnavailable,"not_ready","account service is unavailable"); return }
	var in loginRequest; if err:=decodeJSON(w,r,&in); err!=nil { writeProblem(w,r,http.StatusBadRequest,"invalid_request","invalid JSON body"); return }
	out,err:=s.deps.Accounts.Login(r.Context(),account.Login{Identifier:in.Identifier,Password:in.Password,DeviceLabel:in.DeviceLabel})
	if err != nil {
		if errors.Is(err, account.ErrUnauthorized) {
			writeProblem(w,r,http.StatusUnauthorized,"invalid_credentials","invalid credentials")
		} else {
			writeProblem(w,r,http.StatusServiceUnavailable,"not_ready","authentication service is temporarily unavailable")
		}
		return
	}
	writeJSON(w,http.StatusOK,out)
}

func (s *Server) logout(w http.ResponseWriter,r *http.Request){
	auth,ok:=authFrom(r)
	if !ok { writeProblem(w,r,http.StatusUnauthorized,"unauthorized","authentication required"); return }
	if err:=s.deps.Accounts.Logout(r.Context(),auth.RawToken); err!=nil { writeProblem(w,r,http.StatusInternalServerError,"internal_error","request failed"); return }
	// The session is already revoked at this point, so ActiveTokens cannot use
	// the device even if this cleanup encounters a transient database error.
	if s.deps.Push != nil { _ = s.deps.Push.RevokeSession(r.Context(),auth.SessionID) }
	w.WriteHeader(http.StatusNoContent)
}
func (s *Server) getMe(w http.ResponseWriter,r *http.Request){ auth,_:=authFrom(r); writeJSON(w,http.StatusOK,auth.User) }
func (s *Server) patchMe(w http.ResponseWriter,r *http.Request){ auth,ok:=authFrom(r); if !ok { writeProblem(w,r,http.StatusUnauthorized,"unauthorized","authentication required"); return }; var in patchMeRequest; if err:=decodeJSON(w,r,&in); err!=nil { writeProblem(w,r,http.StatusBadRequest,"invalid_request","invalid JSON body"); return }; out,err:=s.deps.Accounts.UpdateProfile(r.Context(),auth.User.ID,account.ProfilePatch{DisplayName:in.DisplayName,AvatarURL:in.AvatarURL,ProfileVisibility:in.ProfileVisibility,Language:in.Language,Interests:in.Interests}); if err!=nil { s.writeAccountError(w,r,err,true); return }; writeJSON(w,http.StatusOK,out) }

func (s *Server) writeAccountError(w http.ResponseWriter,r *http.Request,err error,allowNotFound bool){ switch { case errors.Is(err,account.ErrInvalidInput): writeProblem(w,r,http.StatusBadRequest,"invalid_request","invalid account data"); case errors.Is(err,account.ErrConflict): writeProblem(w,r,http.StatusConflict,"account_conflict","email or username is unavailable"); case allowNotFound && errors.Is(err,account.ErrNotFound): writeProblem(w,r,http.StatusNotFound,"not_found","account not found"); default: writeProblem(w,r,http.StatusInternalServerError,"internal_error","request failed") } }
