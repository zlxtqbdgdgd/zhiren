package main

import (
	"bytes"
	"html/template"
	"io/fs"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type App struct {
	store  *Store
	layout *template.Template
	pages  map[string]*template.Template
}

var funcMap = template.FuncMap{
	"roleLabel": roleLabel,
	"eq2":       func(a, b string) bool { return a == b },
}

func newApp(store *Store, tplFS fs.FS) (*App, error) {
	app := &App{store: store, pages: map[string]*template.Template{}}
	layout, err := template.New("layout.html").Funcs(funcMap).ParseFS(tplFS, "web/templates/layout.html")
	if err != nil {
		return nil, err
	}
	app.layout = layout
	pageNames := []string{"login", "home", "list", "view", "form", "reminders", "reports", "import", "batch", "admin", "password"}
	for _, n := range pageNames {
		t, err := template.New(n+".html").Funcs(funcMap).ParseFS(tplFS, "web/templates/"+n+".html")
		if err != nil {
			return nil, err
		}
		app.pages[n] = t
	}
	return app, nil
}

type layoutData struct {
	Title string
	User  *User
	Org   string
	Flash string
	Err   string
	Body  template.HTML
}

func (app *App) render(w http.ResponseWriter, r *http.Request, page, title string, user *User, data map[string]any) {
	if data == nil {
		data = map[string]any{}
	}
	data["User"] = user
	data["Fields"] = fields
	var body bytes.Buffer
	if err := app.pages[page].Execute(&body, data); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	ld := layoutData{
		Title: title,
		User:  user,
		Org:   app.store.Settings().OrgName,
		Flash: r.URL.Query().Get("msg"),
		Err:   r.URL.Query().Get("err"),
		Body:  template.HTML(body.String()),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := app.layout.Execute(w, ld); err != nil {
		http.Error(w, err.Error(), 500)
	}
}

func redirect(w http.ResponseWriter, r *http.Request, path, msg, errMsg string) {
	q := url.Values{}
	if msg != "" {
		q.Set("msg", msg)
	}
	if errMsg != "" {
		q.Set("err", errMsg)
	}
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	http.Redirect(w, r, path, http.StatusSeeOther)
}

const cookieName = "zhiren_session"

func (app *App) currentUser(r *http.Request) *User {
	ck, err := r.Cookie(cookieName)
	if err != nil {
		return nil
	}
	u, ok := app.store.UserByToken(ck.Value)
	if !ok {
		return nil
	}
	return u
}

func canEdit(u *User) bool { return u != nil && (u.Role == RoleAdmin || u.Role == RoleMaintainer) }

// ----- 路由 -----

func (app *App) routes(static fs.FS) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(static))))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	mux.HandleFunc("/login", app.handleLogin)
	mux.HandleFunc("/logout", app.handleLogout)

	// 需登录
	mux.HandleFunc("/", app.auth(app.handleHome))
	mux.HandleFunc("/cadres", app.auth(app.handleList))
	mux.HandleFunc("/cadres/view", app.auth(app.handleView))
	mux.HandleFunc("/reminders", app.auth(app.handleReminders))
	mux.HandleFunc("/reminders/export", app.auth(app.handleRemindersExport))
	mux.HandleFunc("/reports", app.auth(app.handleReports))
	mux.HandleFunc("/export", app.auth(app.handleExport))
	mux.HandleFunc("/password", app.auth(app.handlePassword))

	// 需维护权限
	mux.HandleFunc("/cadres/new", app.auth(app.requireEdit(app.handleNew)))
	mux.HandleFunc("/cadres/edit", app.auth(app.requireEdit(app.handleEdit)))
	mux.HandleFunc("/cadres/delete", app.auth(app.requireEdit(app.handleDelete)))
	mux.HandleFunc("/import", app.auth(app.requireEdit(app.handleImport)))
	mux.HandleFunc("/batch", app.auth(app.requireEdit(app.handleBatch)))

	// 需管理员
	mux.HandleFunc("/admin", app.auth(app.requireAdmin(app.handleAdmin)))
	mux.HandleFunc("/admin/users", app.auth(app.requireAdmin(app.handleAdminUsers)))
	mux.HandleFunc("/admin/rules", app.auth(app.requireAdmin(app.handleAdminRules)))
	mux.HandleFunc("/admin/settings", app.auth(app.requireAdmin(app.handleAdminSettings)))
	mux.HandleFunc("/admin/backup", app.auth(app.requireAdmin(app.handleAdminBackup)))
	mux.HandleFunc("/admin/restore", app.auth(app.requireAdmin(app.handleAdminRestore)))
	return mux
}

type handlerWithUser func(http.ResponseWriter, *http.Request, *User)

func (app *App) auth(h handlerWithUser) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := app.currentUser(r)
		if u == nil {
			redirect(w, r, "/login", "", "")
			return
		}
		h(w, r, u)
	}
}

func (app *App) requireEdit(h handlerWithUser) handlerWithUser {
	return func(w http.ResponseWriter, r *http.Request, u *User) {
		if !canEdit(u) {
			redirect(w, r, "/cadres", "", "无权限：仅维护者/管理员可操作")
			return
		}
		h(w, r, u)
	}
}

func (app *App) requireAdmin(h handlerWithUser) handlerWithUser {
	return func(w http.ResponseWriter, r *http.Request, u *User) {
		if u.Role != RoleAdmin {
			redirect(w, r, "/", "", "无权限：仅管理员可操作")
			return
		}
		h(w, r, u)
	}
}

// ----- 视图模型 -----

type CadreView struct {
	*Cadre
	Age       string
	WorkYears string
	Retire    string
}

func (app *App) view(c *Cadre, now time.Time) CadreView {
	set := app.store.Settings()
	return CadreView{
		Cadre:     c,
		Age:       c.AgeText(now, set.AgeWithMonths),
		WorkYears: c.WorkYearsText(now, set.WorkYearsWithDays),
		Retire:    retireDateText(c, app.store.Rules()),
	}
}

func (app *App) views(list []*Cadre, now time.Time) []CadreView {
	out := make([]CadreView, len(list))
	for i, c := range list {
		out[i] = app.view(c, now)
	}
	return out
}

// ----- 登录 -----

func (app *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		_ = r.ParseForm()
		token, _, err := app.store.Login(r.FormValue("username"), r.FormValue("password"))
		if err != nil {
			redirect(w, r, "/login", "", err.Error())
			return
		}
		http.SetCookie(w, &http.Cookie{Name: cookieName, Value: token, Path: "/", HttpOnly: true, MaxAge: int(sessionTTL.Seconds())})
		redirect(w, r, "/", "登录成功", "")
		return
	}
	if app.currentUser(r) != nil {
		redirect(w, r, "/", "", "")
		return
	}
	app.render(w, r, "login", "登录", nil, nil)
}

func (app *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	if ck, err := r.Cookie(cookieName); err == nil {
		app.store.Logout(ck.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: "", Path: "/", MaxAge: -1})
	redirect(w, r, "/login", "已退出", "")
}

// ----- 首页 / 仪表盘 -----

func (app *App) handleHome(w http.ResponseWriter, r *http.Request, u *User) {
	now := time.Now()
	all := app.store.Cadres()
	set := app.store.Settings()
	due := dueCadres(all, app.store.Rules(), now, set.ReminderWindowMonths)
	app.render(w, r, "home", "首页", u, map[string]any{
		"Total":     len(all),
		"Due":       app.views(due, now),
		"DueCount":  len(due),
		"AuditTail": app.store.AuditTail(8),
		"CanEdit":   canEdit(u),
	})
}

// ----- 列表 / 搜索 -----

func (app *App) handleList(w http.ResponseWriter, r *http.Request, u *User) {
	now := time.Now()
	_ = r.ParseForm()
	kw := r.FormValue("q")
	var filters []Filter
	for _, f := range fields {
		v := strings.TrimSpace(r.FormValue("f_" + f.Key))
		if v != "" {
			filters = append(filters, Filter{Key: f.Key, Value: v})
		}
	}
	list := filterCadres(app.store.Cadres(), kw, filters)
	app.render(w, r, "list", "干部查询", u, map[string]any{
		"Q":       kw,
		"Filters": formValues(r),
		"Results": app.views(list, now),
		"Count":   len(list),
		"CanEdit": canEdit(u),
	})
}

func formValues(r *http.Request) map[string]string {
	m := map[string]string{}
	for _, f := range fields {
		m[f.Key] = strings.TrimSpace(r.FormValue("f_" + f.Key))
	}
	return m
}

// ----- 查看 -----

func (app *App) handleView(w http.ResponseWriter, r *http.Request, u *User) {
	c, ok := app.store.Cadre(r.URL.Query().Get("id"))
	if !ok {
		redirect(w, r, "/cadres", "", "记录不存在")
		return
	}
	app.render(w, r, "view", "干部详情", u, map[string]any{
		"C":       app.view(c, time.Now()),
		"CanEdit": canEdit(u),
	})
}

// ----- 新建 / 编辑 -----

func (app *App) handleNew(w http.ResponseWriter, r *http.Request, u *User) {
	if r.Method == http.MethodPost {
		c := parseCadreForm(r)
		if errs := validateCadre(c); len(errs) > 0 {
			app.render(w, r, "form", "新建干部", u, map[string]any{"C": &CadreView{Cadre: c}, "Errors": errs, "New": true})
			return
		}
		if err := app.store.CreateCadre(c, u.Username); err != nil {
			redirect(w, r, "/cadres", "", err.Error())
			return
		}
		redirect(w, r, "/cadres/view?id="+c.ID, "已新建", "")
		return
	}
	app.render(w, r, "form", "新建干部", u, map[string]any{"C": &CadreView{Cadre: &Cadre{F: map[string]string{}}}, "New": true})
}

func (app *App) handleEdit(w http.ResponseWriter, r *http.Request, u *User) {
	if r.Method == http.MethodPost {
		c := parseCadreForm(r)
		c.ID = r.FormValue("id")
		if errs := validateCadre(c); len(errs) > 0 {
			app.render(w, r, "form", "编辑干部", u, map[string]any{"C": &CadreView{Cadre: c}, "Errors": errs})
			return
		}
		if err := app.store.UpdateCadre(c, u.Username); err != nil {
			redirect(w, r, "/cadres", "", err.Error())
			return
		}
		redirect(w, r, "/cadres/view?id="+c.ID, "已保存", "")
		return
	}
	c, ok := app.store.Cadre(r.URL.Query().Get("id"))
	if !ok {
		redirect(w, r, "/cadres", "", "记录不存在")
		return
	}
	app.render(w, r, "form", "编辑干部", u, map[string]any{"C": &CadreView{Cadre: c}})
}

func (app *App) handleDelete(w http.ResponseWriter, r *http.Request, u *User) {
	if r.Method != http.MethodPost {
		redirect(w, r, "/cadres", "", "")
		return
	}
	id := r.FormValue("id")
	if err := app.store.DeleteCadre(id, u.Username); err != nil {
		redirect(w, r, "/cadres", "", err.Error())
		return
	}
	redirect(w, r, "/cadres", "已删除", "")
}

func parseCadreForm(r *http.Request) *Cadre {
	_ = r.ParseForm()
	c := &Cadre{F: map[string]string{}}
	for _, f := range fields {
		c.F[f.Key] = strings.TrimSpace(r.FormValue(f.Key))
	}
	c.RetireOverride = strings.TrimSpace(r.FormValue("retire_override"))
	rel := r.Form["fam_relation"]
	nm := r.Form["fam_name"]
	bd := r.Form["fam_birth"]
	ps := r.Form["fam_political"]
	wp := r.Form["fam_workpos"]
	for i := range rel {
		m := FamilyMember{
			Relation:        strings.TrimSpace(get(rel, i)),
			Name:            strings.TrimSpace(get(nm, i)),
			BirthDate:       strings.TrimSpace(get(bd, i)),
			PoliticalStatus: strings.TrimSpace(get(ps, i)),
			WorkUnitPos:     strings.TrimSpace(get(wp, i)),
		}
		if m.Relation != "" || m.Name != "" {
			c.Family = append(c.Family, m)
		}
	}
	return c
}

func get(s []string, i int) string {
	if i < len(s) {
		return s[i]
	}
	return ""
}

// ----- 到龄提醒 -----

func (app *App) handleReminders(w http.ResponseWriter, r *http.Request, u *User) {
	now := time.Now()
	set := app.store.Settings()
	due := dueCadres(app.store.Cadres(), app.store.Rules(), now, set.ReminderWindowMonths)
	app.render(w, r, "reminders", "到龄提醒", u, map[string]any{
		"Due":    app.views(due, now),
		"Window": set.ReminderWindowMonths,
	})
}

func (app *App) handleRemindersExport(w http.ResponseWriter, r *http.Request, u *User) {
	now := time.Now()
	set := app.store.Settings()
	due := dueCadres(app.store.Cadres(), app.store.Rules(), now, set.ReminderWindowMonths)
	csvBytes := exportCSV(due, app.store.Rules(), now, set)
	writeCSV(w, "到龄名单.csv", csvBytes)
}

// ----- 报表 -----

func (app *App) handleReports(w http.ResponseWriter, r *http.Request, u *User) {
	rep := buildReport(app.store.Cadres(), app.store.Rules(), time.Now())
	app.render(w, r, "reports", "数据分析", u, map[string]any{"Report": rep})
}

// ----- 导出 -----

func (app *App) handleExport(w http.ResponseWriter, r *http.Request, u *User) {
	now := time.Now()
	_ = r.ParseForm()
	kw := r.FormValue("q")
	var filters []Filter
	for _, f := range fields {
		if v := strings.TrimSpace(r.FormValue("f_" + f.Key)); v != "" {
			filters = append(filters, Filter{Key: f.Key, Value: v})
		}
	}
	list := filterCadres(app.store.Cadres(), kw, filters)
	csvBytes := exportCSV(list, app.store.Rules(), now, app.store.Settings())
	writeCSV(w, "干部信息.csv", csvBytes)
}

func writeCSV(w http.ResponseWriter, name string, b []byte) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''"+url.PathEscape(name))
	w.Write(b)
}

// ----- 导入 -----

func (app *App) handleImport(w http.ResponseWriter, r *http.Request, u *User) {
	if r.Method == http.MethodPost {
		f, _, err := r.FormFile("file")
		if err != nil {
			redirect(w, r, "/import", "", "请选择文件")
			return
		}
		defer f.Close()
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(f)
		cadres, warnings, err := importCSV(buf.Bytes())
		if err != nil {
			app.render(w, r, "import", "数据导入", u, map[string]any{"Err2": err.Error()})
			return
		}
		ok := 0
		for _, c := range cadres {
			if err := app.store.CreateCadre(c, u.Username); err == nil {
				ok++
			}
		}
		app.render(w, r, "import", "数据导入", u, map[string]any{
			"Done": true, "OK": ok, "Warnings": warnings,
		})
		return
	}
	app.render(w, r, "import", "数据导入", u, nil)
}

// ----- 批量维护 -----

func (app *App) handleBatch(w http.ResponseWriter, r *http.Request, u *User) {
	now := time.Now()
	_ = r.ParseForm()
	if r.Method == http.MethodPost {
		ids := r.Form["id"]
		key := r.FormValue("key")
		val := r.FormValue("value")
		if key == "" || len(ids) == 0 {
			redirect(w, r, "/batch", "", "请选择人员和要批量设置的字段")
			return
		}
		n, err := app.store.BatchSetField(ids, key, val, u.Username)
		if err != nil {
			redirect(w, r, "/batch", "", err.Error())
			return
		}
		redirect(w, r, "/batch", "已批量更新 "+itoa(n)+" 人", "")
		return
	}
	kw := r.FormValue("q")
	list := filterCadres(app.store.Cadres(), kw, nil)
	// 仅允许批量设置非必填的文本/枚举/日期字段
	var batchFields []FieldDef
	for _, f := range fields {
		if !f.Required {
			batchFields = append(batchFields, f)
		}
	}
	app.render(w, r, "batch", "批量维护", u, map[string]any{
		"Q":           kw,
		"Results":     app.views(list, now),
		"BatchFields": batchFields,
	})
}

// ----- 修改密码 -----

func (app *App) handlePassword(w http.ResponseWriter, r *http.Request, u *User) {
	if r.Method == http.MethodPost {
		err := app.store.ChangeOwnPassword(u.Username, r.FormValue("old"), r.FormValue("new"))
		if err != nil {
			redirect(w, r, "/password", "", err.Error())
			return
		}
		redirect(w, r, "/", "密码已修改", "")
		return
	}
	app.render(w, r, "password", "修改密码", u, nil)
}
