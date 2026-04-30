package dto

type Pagination struct {
	PageIndex int `form:"pageIndex"`
	Page      int `form:"page"` // 兼容前端 page
	PageSize  int `form:"pageSize"`
	PageSizeAlt int `form:"page_size"` // 兼容前端 page_size
}

func (m *Pagination) GetPageIndex() int {
	if m.PageIndex <= 0 && m.Page > 0 {
		m.PageIndex = m.Page
	}
	if m.PageIndex <= 0 {
		m.PageIndex = 1
	}
	return m.PageIndex
}

func (m *Pagination) GetPageSize() int {
	if m.PageSize <= 0 && m.PageSizeAlt > 0 {
		m.PageSize = m.PageSizeAlt
	}
	if m.PageSize <= 0 {
		m.PageSize = 10
	}
	return m.PageSize
}
