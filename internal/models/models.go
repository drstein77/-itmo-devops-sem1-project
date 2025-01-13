package models

type ProcessResponse struct {
	TotalItems      int     `json:"total_items"`
	TotalCategories int     `json:"total_categories"`
	TotalPrice      float64 `json:"total_price"`
}

type Product struct {
	ID        int
	Name      string
	Category  string
	Price     float64
	CreatedAt string
}
