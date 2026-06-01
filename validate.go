package main

import "strings"

// validateCadre 返回所有校验错误（空表示通过）。
func validateCadre(c *Cadre) []string {
	var errs []string
	for _, f := range fields {
		v := strings.TrimSpace(c.get(f.Key))
		if f.Required && v == "" {
			errs = append(errs, f.Label+"不能为空")
			continue
		}
		if v == "" {
			continue
		}
		switch f.Kind {
		case "date":
			if _, ok := parseDate(v); !ok {
				errs = append(errs, f.Label+"日期格式无法识别（示例：2020-03 或 2020-03-15）")
			}
		case "enum":
			if !inList(v, f.Options) {
				errs = append(errs, f.Label+"只能是："+strings.Join(f.Options, "、"))
			}
		}
	}
	if id := strings.TrimSpace(c.get("idcard")); id != "" && !validIDCard(id) {
		errs = append(errs, "身份证号码校验不通过（应为 18 位且校验码正确）")
	}
	if c.RetireOverride != "" {
		if _, ok := parseDate(c.RetireOverride); !ok {
			errs = append(errs, "手工覆盖的到龄时间格式无法识别")
		}
	}
	return errs
}

func inList(v string, list []string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// validIDCard 校验 18 位居民身份证号（含校验码）。
func validIDCard(id string) bool {
	id = strings.ToUpper(strings.TrimSpace(id))
	if len(id) != 18 {
		return false
	}
	weights := []int{7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2}
	checks := "10X98765432"
	sum := 0
	for i := 0; i < 17; i++ {
		ch := id[i]
		if ch < '0' || ch > '9' {
			return false
		}
		sum += int(ch-'0') * weights[i]
	}
	return id[17] == checks[sum%11]
}
