package main

import (
	"sort"
	"strings"
	"time"
)

// Filter 是一条组合筛选条件：字段 Key 的值“精准包含” Value（按字符比对，不做拼音/同音归并）。
type Filter struct {
	Key   string
	Value string
}

func cadreMatchesKeyword(c *Cadre, kw string) bool {
	if kw == "" {
		return true
	}
	for _, v := range c.F {
		if strings.Contains(v, kw) {
			return true
		}
	}
	for _, m := range c.Family {
		if strings.Contains(m.Name, kw) || strings.Contains(m.WorkUnitPos, kw) || strings.Contains(m.Relation, kw) {
			return true
		}
	}
	return strings.Contains(c.RetireOverride, kw)
}

func cadreMatchesFilters(c *Cadre, filters []Filter) bool {
	for _, f := range filters {
		if f.Value == "" {
			continue
		}
		if !strings.Contains(c.get(f.Key), f.Value) {
			return false
		}
	}
	return true
}

// filterCadres 先按全局关键词，再按组合条件（AND），结果按姓名排序。
func filterCadres(list []*Cadre, keyword string, filters []Filter) []*Cadre {
	keyword = strings.TrimSpace(keyword)
	var out []*Cadre
	for _, c := range list {
		if cadreMatchesKeyword(c, keyword) && cadreMatchesFilters(c, filters) {
			out = append(out, c)
		}
	}
	sortByName(out)
	return out
}

func sortByName(list []*Cadre) {
	sort.SliceStable(list, func(i, j int) bool {
		return list[i].get("name") < list[j].get("name")
	})
}

func ym(t time.Time) int { return t.Year()*12 + int(t.Month()) - 1 }

// dueCadres 返回在 [当月, 当月+windowMonths] 内到龄的干部（默认 window=1 即“当月+下月”，
// 对应需求“到龄前一个月及当月”提醒）。
func dueCadres(list []*Cadre, rules []RetireRule, now time.Time, windowMonths int) []*Cadre {
	if windowMonths < 0 {
		windowMonths = 0
	}
	lo := ym(now)
	hi := lo + windowMonths
	var out []*Cadre
	for _, c := range list {
		rd, ok := c.RetireDate(rules)
		if !ok {
			continue
		}
		m := ym(rd)
		if m >= lo && m <= hi {
			out = append(out, c)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		ri, _ := out[i].RetireDate(rules)
		rj, _ := out[j].RetireDate(rules)
		return ri.Before(rj)
	})
	return out
}
