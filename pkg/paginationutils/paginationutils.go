package paginationutils

import (
	"errors"
	"fmt"
	"math"
	"net/url"
)

var ErrInvalidPage = errors.New("invalid page.")

type PaginationView struct {
	// Current page cursor padding
	// Example: I have 10 pages. If I'm on 5 page. With WidthCursorPadding = 2.
	// I will see 3 4 Curr 6 7 pages
	cursorPadding      int
	itemsPerPage       int
	itemsCount         int
	pageQueryParamName string
	url                url.URL
}

type PaginationLink struct {
	Link        string `json:"link"`
	PageNumber  string `json:"page_number"`
	Placeholder bool   `json:"paceholder"`
}

func (p *PaginationView) TotalPages() int {
	return int(math.Ceil(float64(p.itemsCount) / float64(p.itemsPerPage)))
}

func (p *PaginationView) pageLinksRange(start, end int) []PaginationLink {
	var result []PaginationLink
	length := end - start
	for i := 0; i <= length; i++ {
		result = append(result, p.makeLinkFromUrl(i+start))
	}
	return result
}

func (p *PaginationView) PagesLinks(page int) ([]PaginationLink, error) {
	totalPages := p.TotalPages()

	if page > totalPages || page < 1 {
		return nil, errors.Join(ErrInvalidPage, fmt.Errorf("Total pages: %d. Page:%d", totalPages, page))
	}

	isFirstPage := page+p.cursorPadding-totalPages+1 == page

	if p.cursorPadding >= totalPages || isFirstPage {
		allLinks := p.pageLinksRange(1, totalPages)
		return allLinks, nil
	}

	leftBorder := int(math.Max(float64(page-p.cursorPadding), 1))
	rightBorder := int(math.Min(float64(page+p.cursorPadding), float64(totalPages)))

	isLeftBorder := leftBorder < totalPages
	isRightBorder := rightBorder > totalPages-p.cursorPadding

	if isLeftBorder && !isRightBorder {
		if isLeftBorder {
			var result []PaginationLink

			leftSide := page - p.cursorPadding

			if leftSide <= 1 {
				pageOffset := page - 1
				if pageOffset == 0 {
					pageOffset++
				}

				result = append(result, p.pageLinksRange(pageOffset, page+p.cursorPadding)...)
			} else if page-p.cursorPadding-1 == 1 {
				result = append(result, p.pageLinksRange(1, p.cursorPadding)...)
				result = append(result, p.pageLinksRange(leftSide, page+p.cursorPadding)...)
			} else {
				result = append(result, p.pageLinksRange(1, p.cursorPadding)...)
				result = append(result, p.makeLinkPlaceholder())
				result = append(result, p.pageLinksRange(leftSide, page+p.cursorPadding)...)
			}

			rightSide := page + p.cursorPadding + 1

			if rightSide >= totalPages {
				result = append(result, p.makeLinkFromUrl(totalPages))
			} else {
				result = append(result, p.makeLinkPlaceholder())
				result = append(result, p.makeLinkFromUrl(totalPages))
			}

			return result, nil
		}
	}

	rightLinks := p.pageLinksRange(leftBorder, rightBorder)

	leftSideLinks := []PaginationLink{
		p.makeLinkFromUrl(1),
		p.makeLinkPlaceholder(),
	}

	result := append(leftSideLinks, rightLinks...)
	return result, nil
}

func (p *PaginationView) makeLinkFromUrl(page int) PaginationLink {
	queryValues := p.url.Query()
	queryValues.Set(p.pageQueryParamName, fmt.Sprint(page))

	p.url.RawQuery = queryValues.Encode()

	return PaginationLink{
		Link:       p.url.String(),
		PageNumber: fmt.Sprint(page),
	}
}

func (p *PaginationView) makeLinkPlaceholder() PaginationLink {
	return PaginationLink{
		Link:        "...",
		PageNumber:  "...",
		Placeholder: true,
	}
}

type NewPaginationViewParams struct {
	ItemsPerPage       int
	ItemsCount         int
	PageQueryParamName string
}

func NewPaginationView(url url.URL, params NewPaginationViewParams) *PaginationView {
	return &PaginationView{
		url:                url,
		cursorPadding:      1,
		itemsPerPage:       params.ItemsPerPage,
		itemsCount:         params.ItemsCount,
		pageQueryParamName: params.PageQueryParamName,
	}
}
