package main

import (
	"strings"
	"time"
)

// FieldDef 描述一个干部档案的扁平字段，驱动表单渲染、列表列、CSV 表头与校验。
type FieldDef struct {
	Key      string
	Label    string
	Kind     string // text | textarea | date | enum
	Options  []string
	Required bool
}

// fields 是干部档案的扁平字段定义（家庭成员、简历全文检索与计算字段另行处理）。
var fields = []FieldDef{
	{Key: "name", Label: "姓名", Kind: "text", Required: true},
	{Key: "gender", Label: "性别", Kind: "enum", Options: []string{"男", "女"}, Required: true},
	{Key: "birth_date", Label: "出生年月", Kind: "date", Required: true},
	{Key: "ethnicity", Label: "民族", Kind: "text"},
	{Key: "native_place", Label: "籍贯", Kind: "text"},
	{Key: "birthplace", Label: "出生地", Kind: "text"},
	{Key: "party_join", Label: "入党时间", Kind: "date"},
	{Key: "work_start", Label: "参加工作时间", Kind: "date"},
	{Key: "health", Label: "健康状况", Kind: "text"},
	{Key: "prof_title", Label: "专业技术职务", Kind: "text"},
	{Key: "specialty", Label: "熟悉专业有何专长", Kind: "textarea"},
	{Key: "edu_ft", Label: "学历（全日制）", Kind: "text"},
	{Key: "degree_ft", Label: "学位（全日制）", Kind: "text"},
	{Key: "edu_is", Label: "学历（在职）", Kind: "text"},
	{Key: "degree_is", Label: "学位（在职）", Kind: "text"},
	{Key: "school_ft", Label: "毕业院校及专业（全日制）", Kind: "text"},
	{Key: "school_is", Label: "毕业院校及专业（在职）", Kind: "text"},
	{Key: "current_pos", Label: "现任职务", Kind: "text"},
	{Key: "proposed_pos", Label: "拟任职务", Kind: "text"},
	{Key: "proposed_removal", Label: "拟免职务", Kind: "text"},
	{Key: "resume", Label: "简历", Kind: "textarea"},
	{Key: "rewards", Label: "奖惩情况", Kind: "textarea"},
	{Key: "annual_assess", Label: "年度考核情况", Kind: "textarea"},
	{Key: "appoint_reason", Label: "任免理由", Kind: "textarea"},
	{Key: "idcard", Label: "身份证号码", Kind: "text"},
	{Key: "retire_identity", Label: "干部/工人身份", Kind: "enum", Options: []string{"干部", "工人", "其他"}},
	{Key: "personnel_cat", Label: "人员类别", Kind: "text"},
	{Key: "establishment_cat", Label: "编制类别", Kind: "text"},
	{Key: "civil_reg", Label: "公务员登记时间", Kind: "date"},
	{Key: "current_pos_since", Label: "任现职时间", Kind: "date"},
	{Key: "current_rank_since", Label: "任现职务级别时间", Kind: "date"},
	{Key: "age_ref_date", Label: "计算年龄时间", Kind: "date"},
	{Key: "form_date", Label: "填表时间", Kind: "date"},
	{Key: "form_filler", Label: "填表人", Kind: "text"},
}

func fieldDef(key string) (FieldDef, bool) {
	for _, f := range fields {
		if f.Key == key {
			return f, true
		}
	}
	return FieldDef{}, false
}

// FamilyMember 是挂在干部记录下的家庭主要成员及社会关系（可重复子表）。
type FamilyMember struct {
	Relation        string `json:"relation"`         // 称谓
	Name            string `json:"name"`             // 姓名
	BirthDate       string `json:"birth_date"`       // 出生年月
	PoliticalStatus string `json:"political_status"` // 政治面貌
	WorkUnitPos     string `json:"work_unit_pos"`    // 工作单位及职务
}

// Cadre 是一条干部记录。扁平字段统一放在 F（按 FieldDef.Key 索引），
// 便于元数据驱动渲染与导入导出；家庭成员、到龄手工覆盖与时间戳单列。
type Cadre struct {
	ID             string            `json:"id"`
	F              map[string]string `json:"f"`
	Family         []FamilyMember    `json:"family"`
	RetireOverride string            `json:"retire_override"` // 手工覆盖的到龄时间（YYYY-MM-DD），优先于规则计算
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

func (c *Cadre) get(key string) string {
	if c.F == nil {
		return ""
	}
	return c.F[key]
}

// ----- 日期解析与计算字段 -----

// parseDate 容忍多种常见写法：2020-01-02 / 2020/1/2 / 2020.01 / 2020-3 / 2020年3月。
// 缺日按 1 号，缺月按 1 月。返回 ok=false 表示无法解析。
func parseDate(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	r := strings.NewReplacer("年", "-", "月", "-", "日", "", "/", "-", ".", "-", " ", "")
	s = strings.Trim(r.Replace(s), "-")
	parts := strings.Split(s, "-")
	nums := make([]int, 0, 3)
	for _, p := range parts {
		if p == "" {
			continue
		}
		n := 0
		for _, ch := range p {
			if ch < '0' || ch > '9' {
				return time.Time{}, false
			}
			n = n*10 + int(ch-'0')
		}
		nums = append(nums, n)
	}
	if len(nums) == 0 || nums[0] < 1900 || nums[0] > 3000 {
		return time.Time{}, false
	}
	y := nums[0]
	mo := 1
	d := 1
	if len(nums) >= 2 {
		mo = nums[1]
	}
	if len(nums) >= 3 {
		d = nums[2]
	}
	if mo < 1 || mo > 12 || d < 1 || d > 31 {
		return time.Time{}, false
	}
	return time.Date(y, time.Month(mo), d, 0, 0, 0, 0, time.Local), true
}

// diffYMD 返回 from 到 to 的整年、整月、整日差（to>=from 时为非负）。
func diffYMD(from, to time.Time) (years, months, days int) {
	if to.Before(from) {
		return 0, 0, 0
	}
	years = to.Year() - from.Year()
	months = int(to.Month()) - int(from.Month())
	days = to.Day() - from.Day()
	if days < 0 {
		months--
		// 借上一个月的天数
		prev := time.Date(to.Year(), to.Month(), 0, 0, 0, 0, 0, time.Local)
		days += prev.Day()
	}
	if months < 0 {
		years--
		months += 12
	}
	return years, months, days
}

// ageRefDate 返回该记录用于计算年龄的基准日：优先字段“计算年龄时间”，否则今天。
func (c *Cadre) ageRefDate(now time.Time) time.Time {
	if t, ok := parseDate(c.get("age_ref_date")); ok {
		return t
	}
	return now
}

// AgeYears 周岁；ok=false 表示出生年月无效。
func (c *Cadre) AgeYears(now time.Time) (int, bool) {
	b, ok := parseDate(c.get("birth_date"))
	if !ok {
		return 0, false
	}
	y, _, _ := diffYMD(b, c.ageRefDate(now))
	return y, true
}

// AgeText 输出“XX岁”或“XX岁XX个月”。
func (c *Cadre) AgeText(now time.Time, withMonths bool) string {
	b, ok := parseDate(c.get("birth_date"))
	if !ok {
		return ""
	}
	y, m, _ := diffYMD(b, c.ageRefDate(now))
	if withMonths {
		return itoa(y) + "岁" + itoa(m) + "个月"
	}
	return itoa(y) + "岁"
}

// WorkYearsText 工作年限，精确到年月日。
func (c *Cadre) WorkYearsText(now time.Time, withDays bool) string {
	w, ok := parseDate(c.get("work_start"))
	if !ok {
		return ""
	}
	y, m, d := diffYMD(w, now)
	if withDays {
		return itoa(y) + "年" + itoa(m) + "个月" + itoa(d) + "天"
	}
	return itoa(y) + "年" + itoa(m) + "个月"
}

// RetireDate 计算到龄时间：优先手工覆盖，否则按退休规则（出生年月 + 退休年龄）。
func (c *Cadre) RetireDate(rules []RetireRule) (time.Time, bool) {
	if t, ok := parseDate(c.RetireOverride); ok {
		return t, true
	}
	b, ok := parseDate(c.get("birth_date"))
	if !ok {
		return time.Time{}, false
	}
	age := retireAge(rules, c.get("gender"), c.get("retire_identity"))
	if age <= 0 {
		return time.Time{}, false
	}
	return b.AddDate(age, 0, 0), true
}

// itoa 极简整数转字符串（避免引入 strconv 仅为此）。
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
