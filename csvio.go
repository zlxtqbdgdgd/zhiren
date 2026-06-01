package main

import (
	"bytes"
	"encoding/csv"
	"strings"
	"time"
)

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// exportCSV 导出为带 BOM 的 UTF-8 CSV，Excel 可直接打开且中文不乱码。
func exportCSV(list []*Cadre, rules []RetireRule, now time.Time, set Settings) []byte {
	var buf bytes.Buffer
	buf.Write(utf8BOM)
	w := csv.NewWriter(&buf)

	header := []string{"编号"}
	for _, f := range fields {
		header = append(header, f.Label)
	}
	header = append(header, "年龄", "工作年限", "到龄时间", "家庭成员及社会关系")
	_ = w.Write(header)

	for _, c := range list {
		row := []string{c.ID}
		for _, f := range fields {
			row = append(row, c.get(f.Key))
		}
		row = append(row, c.AgeText(now, set.AgeWithMonths), c.WorkYearsText(now, set.WorkYearsWithDays), retireDateText(c, rules), familySummary(c))
		_ = w.Write(row)
	}
	w.Flush()
	return buf.Bytes()
}

func retireDateText(c *Cadre, rules []RetireRule) string {
	if t, ok := c.RetireDate(rules); ok {
		return t.Format("2006-01-02")
	}
	return ""
}

func familySummary(c *Cadre) string {
	var parts []string
	for _, m := range c.Family {
		parts = append(parts, m.Relation+":"+m.Name+"("+m.PoliticalStatus+","+m.WorkUnitPos+")")
	}
	return strings.Join(parts, "；")
}

// importCSV 解析 CSV（按表头标签或字段 Key 映射到字段），返回待新建的干部与告警。
func importCSV(b []byte) ([]*Cadre, []string, error) {
	b = bytes.TrimPrefix(b, utf8BOM)
	r := csv.NewReader(bytes.NewReader(b))
	r.FieldsPerRecord = -1
	records, err := r.ReadAll()
	if err != nil {
		return nil, nil, err
	}
	if len(records) < 2 {
		return nil, []string{"文件为空或只有表头"}, nil
	}

	labelToKey := map[string]string{}
	for _, f := range fields {
		labelToKey[f.Label] = f.Key
		labelToKey[f.Key] = f.Key
	}

	header := records[0]
	colKey := make([]string, len(header))
	for i, h := range header {
		colKey[i] = labelToKey[strings.TrimSpace(strings.TrimPrefix(h, string(utf8BOM)))]
	}

	var out []*Cadre
	var warnings []string
	for ri, rec := range records[1:] {
		c := &Cadre{F: map[string]string{}}
		empty := true
		for i, val := range rec {
			if i >= len(colKey) || colKey[i] == "" {
				continue
			}
			val = strings.TrimSpace(val)
			if val != "" {
				c.F[colKey[i]] = val
				empty = false
			}
		}
		if empty {
			continue
		}
		if errs := validateCadre(c); len(errs) > 0 {
			warnings = append(warnings, "第"+itoa(ri+2)+"行（"+c.get("name")+"）："+strings.Join(errs, "；"))
			continue
		}
		out = append(out, c)
	}
	return out, warnings, nil
}
