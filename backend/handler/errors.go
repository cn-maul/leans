package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// errHTTP maps a domain error to an HTTP status + message. Define sentinel
// errors in the packages that own them so handlers stay thin.
type errHTTP struct {
	status int
	msg    string
}

func (e *errHTTP) Error() string { return e.msg }

// NewErrHTTP builds a domain error that respondError converts into a JSON body.
func NewErrHTTP(status int, msg string) error {
	return &errHTTP{status: status, msg: msg}
}

// respondError writes a unified error envelope: {"error": "..."}.
func respondError(c *gin.Context, err error) {
	var he *errHTTP
	if errors.As(err, &he) {
		c.JSON(he.status, gin.H{"error": he.msg})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}
