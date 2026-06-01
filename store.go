package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// 角色常量
const (
	RoleAdmin      = "admin"      // 管理员：管账号与配置
	RoleMaintainer = "maintainer" // 维护者：干部信息增删改（干部组）
	RoleViewer     = "viewer"     // 查看者：只读+搜索+导出
)

func roleLabel(r string) string {
	switch r {
	case RoleAdmin:
		return "管理员"
	case RoleMaintainer:
		return "维护者"
	case RoleViewer:
		return "查看者"
	}
	return r
}

type User struct {
	Username string `json:"username"`
	Display  string `json:"display"`
	Role     string `json:"role"`
	Salt     string `json:"salt"`
	Hash     string `json:"hash"`
	Disabled bool   `json:"disabled"`
}

// RetireRule 退休年龄规则：性别×身份 → 退休周岁。空串表示通配。
type RetireRule struct {
	Gender   string `json:"gender"`
	Identity string `json:"identity"`
	AgeYears int    `json:"age_years"`
}

type AuditEntry struct {
	Time   time.Time `json:"time"`
	User   string    `json:"user"`
	Action string    `json:"action"`
	Target string    `json:"target"`
	Detail string    `json:"detail"`
}

type Settings struct {
	OrgName              string `json:"org_name"`
	AgeWithMonths        bool   `json:"age_with_months"`
	WorkYearsWithDays    bool   `json:"work_years_with_days"`
	BackupDir            string `json:"backup_dir"`
	BackupKeep           int    `json:"backup_keep"`
	BackupEveryHours     int    `json:"backup_every_hours"`
	ReminderWindowMonths int    `json:"reminder_window_months"`
}

// Data 是持久化的根对象。
type Data struct {
	Cadres   []*Cadre     `json:"cadres"`
	Users    []*User      `json:"users"`
	Rules    []RetireRule `json:"rules"`
	Audit    []AuditEntry `json:"audit"`
	Settings Settings     `json:"settings"`
	Seq      int          `json:"seq"`
}

type session struct {
	username string
	expiry   time.Time
}

// Store 是带读写锁的内存数据 + JSON 原子落盘。
type Store struct {
	mu       sync.RWMutex
	data     *Data
	path     string
	sessions map[string]session
}

func defaultSettings() Settings {
	return Settings{
		OrgName:              "本单位",
		AgeWithMonths:        false,
		WorkYearsWithDays:    false,
		BackupDir:            "data/backups",
		BackupKeep:           30,
		BackupEveryHours:     24,
		ReminderWindowMonths: 1,
	}
}

func defaultRules() []RetireRule {
	return []RetireRule{
		{Gender: "男", Identity: "", AgeYears: 60},
		{Gender: "女", Identity: "干部", AgeYears: 55},
		{Gender: "女", Identity: "工人", AgeYears: 50},
		{Gender: "女", Identity: "", AgeYears: 55},
	}
}

// retireAge 选最具体的匹配规则（匹配的非空条件越多越优先）。
func retireAge(rules []RetireRule, gender, identity string) int {
	best, bestScore := 0, -1
	for _, r := range rules {
		if r.Gender != "" && r.Gender != gender {
			continue
		}
		if r.Identity != "" && r.Identity != identity {
			continue
		}
		score := 0
		if r.Gender != "" {
			score++
		}
		if r.Identity != "" {
			score++
		}
		if score > bestScore {
			best, bestScore = r.AgeYears, score
		}
	}
	return best
}

func NewStore(path string) (*Store, error) {
	s := &Store{path: path, sessions: map[string]session{}}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		s.data = &Data{Rules: defaultRules(), Settings: defaultSettings()}
		s.data.Settings.BackupDir = filepath.Join(filepath.Dir(path), "backups")
		admin := &User{Username: "admin", Display: "管理员", Role: RoleAdmin}
		setPassword(admin, "admin123")
		s.data.Users = append(s.data.Users, admin)
		if err := s.save(); err != nil {
			return nil, err
		}
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	var d Data
	if err := json.Unmarshal(b, &d); err != nil {
		return nil, err
	}
	if len(d.Rules) == 0 {
		d.Rules = defaultRules()
	}
	if d.Settings.BackupKeep == 0 {
		d.Settings = defaultSettings()
	}
	s.data = &d
	return s, nil
}

// save 原子写入（调用方需持有写锁）。
func (s *Store) save() error {
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// ----- 设置与规则 -----

func (s *Store) Settings() Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.Settings
}

func (s *Store) SetSettings(set Settings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Settings = set
	return s.save()
}

func (s *Store) Rules() []RetireRule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]RetireRule, len(s.data.Rules))
	copy(out, s.data.Rules)
	return out
}

func (s *Store) SetRules(rules []RetireRule) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Rules = rules
	return s.save()
}

// ----- 干部 CRUD -----

func (s *Store) Cadres() []*Cadre {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Cadre, len(s.data.Cadres))
	copy(out, s.data.Cadres)
	return out
}

func (s *Store) Cadre(id string) (*Cadre, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.data.Cadres {
		if c.ID == id {
			return c, true
		}
	}
	return nil, false
}

func (s *Store) CreateCadre(c *Cadre, user string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Seq++
	c.ID = "C" + pad4(s.data.Seq)
	now := time.Now()
	c.CreatedAt, c.UpdatedAt = now, now
	s.data.Cadres = append(s.data.Cadres, c)
	s.audit(user, "新建干部", c.ID, c.get("name"))
	return s.save()
}

func (s *Store) UpdateCadre(c *Cadre, user string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, old := range s.data.Cadres {
		if old.ID == c.ID {
			c.CreatedAt = old.CreatedAt
			c.UpdatedAt = time.Now()
			s.data.Cadres[i] = c
			s.audit(user, "修改干部", c.ID, c.get("name"))
			return s.save()
		}
	}
	return errors.New("记录不存在")
}

func (s *Store) DeleteCadre(id, user string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, c := range s.data.Cadres {
		if c.ID == id {
			s.data.Cadres = append(s.data.Cadres[:i], s.data.Cadres[i+1:]...)
			s.audit(user, "删除干部", id, c.get("name"))
			return s.save()
		}
	}
	return errors.New("记录不存在")
}

// BatchSetField 批量设置某字段（用于年度考核、公务员转正等批量维护）。
func (s *Store) BatchSetField(ids []string, key, value, user string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	idset := map[string]bool{}
	for _, id := range ids {
		idset[id] = true
	}
	n := 0
	for _, c := range s.data.Cadres {
		if idset[c.ID] {
			if c.F == nil {
				c.F = map[string]string{}
			}
			c.F[key] = value
			c.UpdatedAt = time.Now()
			n++
		}
	}
	if n > 0 {
		s.audit(user, "批量维护", key, value+"（"+itoa(n)+"人）")
		if err := s.save(); err != nil {
			return 0, err
		}
	}
	return n, nil
}

// ----- 审计 -----

func (s *Store) audit(user, action, target, detail string) {
	s.data.Audit = append(s.data.Audit, AuditEntry{
		Time: time.Now(), User: user, Action: action, Target: target, Detail: detail,
	})
	// 仅保留最近 5000 条
	if len(s.data.Audit) > 5000 {
		s.data.Audit = s.data.Audit[len(s.data.Audit)-5000:]
	}
}

func (s *Store) AuditTail(n int) []AuditEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a := s.data.Audit
	if len(a) > n {
		a = a[len(a)-n:]
	}
	out := make([]AuditEntry, len(a))
	copy(out, a)
	// 逆序：最新在前
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// ----- 备份 -----

func (s *Store) Backup() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.backupLocked()
}

func (s *Store) backupLocked() (string, error) {
	set := s.data.Settings
	if err := os.MkdirAll(set.BackupDir, 0o755); err != nil {
		return "", err
	}
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return "", err
	}
	name := "zhiren-" + time.Now().Format("20060102-150405") + ".json"
	full := filepath.Join(set.BackupDir, name)
	if err := os.WriteFile(full, b, 0o644); err != nil {
		return "", err
	}
	s.pruneBackups(set)
	return name, nil
}

func (s *Store) pruneBackups(set Settings) {
	if set.BackupKeep <= 0 {
		return
	}
	names := s.listBackupNames(set.BackupDir)
	if len(names) <= set.BackupKeep {
		return
	}
	for _, n := range names[:len(names)-set.BackupKeep] {
		_ = os.Remove(filepath.Join(set.BackupDir, n))
	}
}

func (s *Store) listBackupNames(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names) // 文件名内含时间戳，字典序即时间序
	return names
}

func (s *Store) ListBackups() []string {
	s.mu.RLock()
	dir := s.data.Settings.BackupDir
	s.mu.RUnlock()
	names := s.listBackupNames(dir)
	// 逆序：最新在前
	for i, j := 0, len(names)-1; i < j; i, j = i+1, j-1 {
		names[i], names[j] = names[j], names[i]
	}
	return names
}

func (s *Store) Restore(name, user string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	full := filepath.Join(s.data.Settings.BackupDir, filepath.Base(name))
	b, err := os.ReadFile(full)
	if err != nil {
		return err
	}
	var d Data
	if err := json.Unmarshal(b, &d); err != nil {
		return errors.New("备份文件损坏或格式不符")
	}
	s.data = &d
	s.audit(user, "恢复备份", name, "")
	return s.save()
}

// StartBackupScheduler 后台定时备份。
func (s *Store) StartBackupScheduler() {
	go func() {
		set := s.Settings()
		every := set.BackupEveryHours
		if every <= 0 {
			every = 24
		}
		ticker := time.NewTicker(time.Duration(every) * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			_, _ = s.Backup()
		}
	}()
}

func pad4(n int) string {
	s := itoa(n)
	for len(s) < 4 {
		s = "0" + s
	}
	return s
}
