package main

import (
	"sort"
	"time"
)

type Bucket struct {
	Label string
	Count int
}

type ReportSection struct {
	Title   string
	Buckets []Bucket
}

type Report struct {
	Total    int
	Sections []ReportSection
}

func countBy(list []*Cadre, keyFn func(*Cadre) string) []Bucket {
	m := map[string]int{}
	var order []string
	for _, c := range list {
		k := keyFn(c)
		if k == "" {
			k = "（未填）"
		}
		if _, ok := m[k]; !ok {
			order = append(order, k)
		}
		m[k]++
	}
	sort.SliceStable(order, func(i, j int) bool { return m[order[i]] > m[order[j]] })
	out := make([]Bucket, 0, len(order))
	for _, k := range order {
		out = append(out, Bucket{Label: k, Count: m[k]})
	}
	return out
}

func ageBucket(age int) string {
	switch {
	case age < 30:
		return "30岁以下"
	case age < 40:
		return "30-39岁"
	case age < 50:
		return "40-49岁"
	case age < 55:
		return "50-54岁"
	case age < 60:
		return "55-59岁"
	default:
		return "60岁及以上"
	}
}

func buildReport(list []*Cadre, rules []RetireRule, now time.Time) Report {
	rep := Report{Total: len(list)}

	rep.Sections = append(rep.Sections, ReportSection{
		Title:   "性别分布",
		Buckets: countBy(list, func(c *Cadre) string { return c.get("gender") }),
	})

	rep.Sections = append(rep.Sections, ReportSection{
		Title: "年龄结构",
		Buckets: orderedAgeBuckets(countBy(list, func(c *Cadre) string {
			if a, ok := c.AgeYears(now); ok {
				return ageBucket(a)
			}
			return "（未填）"
		})),
	})

	rep.Sections = append(rep.Sections, ReportSection{
		Title:   "学历分布（全日制）",
		Buckets: countBy(list, func(c *Cadre) string { return c.get("edu_ft") }),
	})

	rep.Sections = append(rep.Sections, ReportSection{
		Title:   "编制类别分布",
		Buckets: countBy(list, func(c *Cadre) string { return c.get("establishment_cat") }),
	})

	rep.Sections = append(rep.Sections, ReportSection{
		Title:   "人员类别分布",
		Buckets: countBy(list, func(c *Cadre) string { return c.get("personnel_cat") }),
	})

	rep.Sections = append(rep.Sections, ReportSection{
		Title: "未来五年到龄分布",
		Buckets: orderedYearBuckets(now, countBy(list, func(c *Cadre) string {
			rd, ok := c.RetireDate(rules)
			if !ok {
				return "（未知）"
			}
			return itoa(rd.Year()) + "年"
		})),
	})

	return rep
}

// orderedAgeBuckets 让年龄段按固定顺序展示。
func orderedAgeBuckets(b []Bucket) []Bucket {
	order := []string{"30岁以下", "30-39岁", "40-49岁", "50-54岁", "55-59岁", "60岁及以上", "（未填）"}
	return reorder(b, order)
}

func orderedYearBuckets(now time.Time, b []Bucket) []Bucket {
	var order []string
	for i := 0; i < 5; i++ {
		order = append(order, itoa(now.Year()+i)+"年")
	}
	return reorder(b, order)
}

func reorder(b []Bucket, order []string) []Bucket {
	idx := map[string]int{}
	for i, k := range order {
		idx[k] = i
	}
	sort.SliceStable(b, func(i, j int) bool {
		oi, iok := idx[b[i].Label]
		oj, jok := idx[b[j].Label]
		if iok && jok {
			return oi < oj
		}
		if iok != jok {
			return iok
		}
		return b[i].Label < b[j].Label
	})
	return b
}
