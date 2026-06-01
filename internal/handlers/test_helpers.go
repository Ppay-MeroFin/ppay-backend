package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

func writeJSONError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{
		Code:    code,
		Message: message,
	})
}

type sqlNullString struct {
	sql.NullString
}

func (ns *sqlNullString) Scan(value any) error {
	switch v := value.(type) {
	case nil:
		ns.String = ""
		ns.Valid = false
		return nil
	case string:
		ns.String = v
		ns.Valid = true
		return nil
	case []byte:
		ns.String = string(v)
		ns.Valid = true
		return nil
	default:
		ns.String = ""
		ns.Valid = false
		return nil
	}
}

type sqlNullTime struct {
	sql.NullTime
}

func (nt *sqlNullTime) Scan(value any) error {
	switch v := value.(type) {
	case nil:
		nt.Time = time.Time{}
		nt.Valid = false
		return nil
	case time.Time:
		nt.Time = v
		nt.Valid = true
		return nil
	default:
		nt.Time = time.Time{}
		nt.Valid = false
		return nil
	}
}

func (nt sqlNullTime) Ptr() *time.Time {
	if !nt.Valid {
		return nil
	}
	t := nt.Time
	return &t
}
