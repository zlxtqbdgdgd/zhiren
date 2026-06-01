package main

import (
	"net/http"
	"strconv"
)

func atoi(s string, def int) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

func (app *App) handleAdmin(w http.ResponseWriter, r *http.Request, u *User) {
	app.render(w, r, "admin", "系统管理", u, map[string]any{
		"Users":      app.store.Users(),
		"Rules":      app.store.Rules(),
		"Settings":   app.store.Settings(),
		"Backups":    app.store.ListBackups(),
		"AuditTail":  app.store.AuditTail(50),
		"Genders":    []string{"", "男", "女"},
		"Identities": []string{"", "干部", "工人", "其他"},
		"Roles":      []string{RoleMaintainer, RoleViewer, RoleAdmin},
	})
}

func (app *App) handleAdminUsers(w http.ResponseWriter, r *http.Request, u *User) {
	if r.Method != http.MethodPost {
		redirect(w, r, "/admin", "", "")
		return
	}
	_ = r.ParseForm()
	action := r.FormValue("action")
	username := r.FormValue("username")
	var err error
	switch action {
	case "create":
		err = app.store.CreateUser(username, r.FormValue("display"), r.FormValue("role"), r.FormValue("password"), u.Username)
	case "update":
		err = app.store.UpdateUser(username, r.FormValue("display"), r.FormValue("role"), r.FormValue("disabled") == "1", r.FormValue("password"), u.Username)
	case "delete":
		err = app.store.DeleteUser(username, u.Username)
	}
	if err != nil {
		redirect(w, r, "/admin", "", err.Error())
		return
	}
	redirect(w, r, "/admin", "账号已更新", "")
}

func (app *App) handleAdminRules(w http.ResponseWriter, r *http.Request, u *User) {
	if r.Method != http.MethodPost {
		redirect(w, r, "/admin", "", "")
		return
	}
	_ = r.ParseForm()
	genders := r.Form["rule_gender"]
	identities := r.Form["rule_identity"]
	ages := r.Form["rule_age"]
	var rules []RetireRule
	for i := range ages {
		age := atoi(get(ages, i), 0)
		if age <= 0 {
			continue
		}
		rules = append(rules, RetireRule{
			Gender:   get(genders, i),
			Identity: get(identities, i),
			AgeYears: age,
		})
	}
	if err := app.store.SetRules(rules); err != nil {
		redirect(w, r, "/admin", "", err.Error())
		return
	}
	redirect(w, r, "/admin", "退休规则已保存", "")
}

func (app *App) handleAdminSettings(w http.ResponseWriter, r *http.Request, u *User) {
	if r.Method != http.MethodPost {
		redirect(w, r, "/admin", "", "")
		return
	}
	_ = r.ParseForm()
	set := app.store.Settings()
	set.OrgName = r.FormValue("org_name")
	set.AgeWithMonths = r.FormValue("age_with_months") == "1"
	set.WorkYearsWithDays = r.FormValue("work_years_with_days") == "1"
	set.BackupDir = r.FormValue("backup_dir")
	set.BackupKeep = atoi(r.FormValue("backup_keep"), 30)
	set.BackupEveryHours = atoi(r.FormValue("backup_every_hours"), 24)
	set.ReminderWindowMonths = atoi(r.FormValue("reminder_window_months"), 1)
	if err := app.store.SetSettings(set); err != nil {
		redirect(w, r, "/admin", "", err.Error())
		return
	}
	redirect(w, r, "/admin", "设置已保存", "")
}

func (app *App) handleAdminBackup(w http.ResponseWriter, r *http.Request, u *User) {
	name, err := app.store.Backup()
	if err != nil {
		redirect(w, r, "/admin", "", err.Error())
		return
	}
	redirect(w, r, "/admin", "已备份："+name, "")
}

func (app *App) handleAdminRestore(w http.ResponseWriter, r *http.Request, u *User) {
	if r.Method != http.MethodPost {
		redirect(w, r, "/admin", "", "")
		return
	}
	name := r.FormValue("name")
	if err := app.store.Restore(name, u.Username); err != nil {
		redirect(w, r, "/admin", "", err.Error())
		return
	}
	redirect(w, r, "/admin", "已从备份恢复："+name, "")
}
