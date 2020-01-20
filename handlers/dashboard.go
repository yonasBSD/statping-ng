// Statping
// Copyright (C) 2018.  Hunter Long and the project contributors
// Written by Hunter Long <info@socialeck.com> and the project contributors
//
// https://github.com/hunterlong/statping
//
// The licenses for most software and other practical works are designed
// to take away your freedom to share and change the works.  By contrast,
// the GNU General Public License is intended to guarantee your freedom to
// share and change all versions of a program--to make sure it remains free
// software for all its users.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package handlers

import (
	"bytes"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/hunterlong/statping/core"
	"github.com/hunterlong/statping/core/notifier"
	"github.com/hunterlong/statping/source"
	"github.com/hunterlong/statping/utils"
	"net/http"
	"strconv"
	"time"
)

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	if !IsUser(r) {
		err := core.ErrorResponse{}
		ExecuteResponse(w, r, "login.gohtml", err, nil)
	} else {
		ExecuteResponse(w, r, "dashboard.gohtml", core.CoreApp, nil)
	}
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	form := parseForm(r)
	username := form.Get("username")
	password := form.Get("password")
	user, auth := core.AuthUser(username, password)
	if auth {
		claim, stt := setJwtToken(user, w)
		fmt.Println(claim.Username, stt)
		utils.Log.Infoln(fmt.Sprintf("User %v logged in from IP %v", user.Username, r.RemoteAddr))
		http.Redirect(w, r, basePath+"dashboard", http.StatusSeeOther)
	} else {
		err := core.ErrorResponse{Error: "Incorrect login information submitted, try again."}
		ExecuteResponse(w, r, "login.gohtml", err, nil)
	}
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	removeJwtToken(w)
	http.Redirect(w, r, basePath, http.StatusSeeOther)
}

func helpHandler(w http.ResponseWriter, r *http.Request) {
	if !IsUser(r) {
		http.Redirect(w, r, basePath, http.StatusSeeOther)
		return
	}
	help := source.HelpMarkdown()
	ExecuteResponse(w, r, "help.gohtml", help, nil)
}

func logsHandler(w http.ResponseWriter, r *http.Request) {
	utils.LockLines.Lock()
	logs := make([]string, 0)
	length := len(utils.LastLines)
	// We need string log lines from end to start.
	for i := length - 1; i >= 0; i-- {
		logs = append(logs, utils.LastLines[i].FormatForHtml()+"\r\n")
	}
	utils.LockLines.Unlock()
	ExecuteResponse(w, r, "logs.gohtml", logs, nil)
}

func logsLineHandler(w http.ResponseWriter, r *http.Request) {
	if lastLine := utils.GetLastLine(); lastLine != nil {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(lastLine.FormatForHtml()))
	}
}

func exportHandler(w http.ResponseWriter, r *http.Request) {
	var notifiers []*notifier.Notification
	for _, v := range core.CoreApp.Notifications {
		notifier := v.(notifier.Notifier)
		notifiers = append(notifiers, notifier.Select())
	}

	export, _ := core.ExportSettings()

	mime := http.DetectContentType(export)
	fileSize := len(string(export))

	w.Header().Set("Content-Type", mime)
	w.Header().Set("Content-Disposition", "attachment; filename=export.json")
	w.Header().Set("Expires", "0")
	w.Header().Set("Content-Transfer-Encoding", "binary")
	w.Header().Set("Content-Length", strconv.Itoa(fileSize))
	w.Header().Set("Content-Control", "private, no-transform, no-store, must-revalidate")

	http.ServeContent(w, r, "export.json", utils.Now(), bytes.NewReader(export))

}

type JwtClaim struct {
	Username string `json:"username"`
	Admin    bool   `json:"admin"`
	jwt.StandardClaims
}

func removeJwtToken(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:    cookieKey,
		Value:   "",
		Expires: time.Now().UTC(),
	})
}

func setJwtToken(user *core.User, w http.ResponseWriter) (JwtClaim, string) {
	expirationTime := time.Now().Add(72 * time.Hour)
	jwtClaim := JwtClaim{
		Username: user.Username,
		Admin:    user.Admin.Bool,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		}}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtClaim)
	tokenString, err := token.SignedString([]byte(jwtKey))
	if err != nil {
		utils.Log.Errorln("error setting token: ", err)
	}
	http.SetCookie(w, &http.Cookie{
		Name:    cookieKey,
		Value:   tokenString,
		Expires: expirationTime,
	})
	return jwtClaim, tokenString
}

func apiLoginHandler(w http.ResponseWriter, r *http.Request) {
	form := parseForm(r)
	username := form.Get("username")
	password := form.Get("password")
	user, auth := core.AuthUser(username, password)
	if auth {
		utils.Log.Infoln(fmt.Sprintf("User %v logged in from IP %v", user.Username, r.RemoteAddr))
		_, token := setJwtToken(user, w)

		resp := struct {
			Token string `json:"token"`
		}{
			token,
		}
		returnJson(resp, w, r)
	} else {
		resp := struct {
			Error string `json:"error"`
		}{
			"incorrect authentication",
		}
		returnJson(resp, w, r)
	}
}
