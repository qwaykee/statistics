package statistics

import (
	"github.com/gin-gonic/gin"

	"time"
	"sync"
	"slices"
	"cmp"
	"fmt"
	"regexp"
	"strings"
)

type (
	Statistics struct {
		Visitors map[string]*Visitor
		Pages map[string]*Page
		Visits map[int]*Visit
		VisitorsLanguage map[string]int

		currentVisitID int
		mutex sync.Mutex
	}

	Page struct {
		Path string
		Visits []*Visit
	}

	Visitor struct {
		IP string
		Language string
		DynamicVisits int
		StaticVisits int
		History []*Visit
	}

	Visit struct {
		ID int
		Date time.Time
		Type PageType
		LoadingTime time.Duration
		TimeSpent time.Duration
		CodeIssued int
		ContentType string
		Referer string
		VisitedBy *Visitor
		Page *Page
	}

	pagesSlice []*Page

	PageType string
)

const (
	Dynamic PageType = "route"
	Static PageType = "static"
)

func New() *Statistics {
	return &Statistics{
		Pages: make(map[string]*Page),
		Visitors: make(map[string]*Visitor),
		Visits: make(map[int]*Visit),
		VisitorsLanguage: make(map[string]int),
	}
}

func containsAny(s string, substrings ...string) bool {
	for _, substr := range substrings {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}

func hasAnySuffix(s string, suffixes ...string) bool {
	for _, suffix := range suffixes {
		if strings.HasSuffix(s, suffix) {
			return true
		}
	}
	return false
}

func (s *Statistics) Middleware() gin.HandlerFunc {
	acceptLanguageRe := regexp.MustCompile(`([a-z]{2});`)

	return func(c *gin.Context) {
		s.mutex.Lock()

		s.currentVisitID++
		c.Set("VisitID", s.currentVisitID)

		s.mutex.Unlock()

		start := time.Now()

		c.Next()

		loadingTime := time.Since(start)

		pagePath := c.Request.URL.Path
		visitorIP := c.ClientIP()

		s.mutex.Lock()
		defer s.mutex.Unlock()

		if _, ok := s.Pages[pagePath]; !ok {
			s.Pages[pagePath] = &Page{}
		}

		if _, ok:= s.Visitors[visitorIP]; !ok {
			lang := c.GetHeader("Accept-Language")

			s.Visitors[visitorIP] = &Visitor{
				IP: c.ClientIP(),
				Language: lang,
			}

			for _, l := range acceptLanguageRe.FindAllStringSubmatch(lang, -1) {
				s.VisitorsLanguage[l[1]] = s.VisitorsLanguage[l[1]] + 1
			}

		}

		visitor := s.Visitors[visitorIP]
		page := s.Pages[pagePath]

		if len(visitor.History) > 1 {
			lastHTMLVisit := visitor.LastDynamicVisit()

			lastHTMLVisit.TimeSpent = time.Since(lastHTMLVisit.Date)
		}

		// determine page type
		contentType := c.Writer.Header().Get("Content-Type")

		var pageType PageType

		pT, exists := c.Get("PageType")

		if pT2, ok := pT.(PageType); exists && ok {
			pageType = pT2
		} else if strings.Contains(contentType, "text/html") {
			pageType = Dynamic
		} else {
			pageType = Static
		}

		if pageType == Dynamic { visitor.DynamicVisits += 1 }
		if pageType == Static { visitor.StaticVisits += 1 }
		

		visit := &Visit{
			ID: s.currentVisitID,
			Type: pageType,
			Date: time.Now(),
			TimeSpent: 0,
			Referer: c.GetHeader("Referer"),
			ContentType: contentType,
			CodeIssued: c.Writer.Status(),
			LoadingTime: loadingTime,
			VisitedBy: visitor,
			Page: page,
		}

		page.Visits = append(page.Visits, visit)
		visitor.History = append(visitor.History, visit)
		s.Visits[s.currentVisitID] = visit
	}
}

func (s *Statistics) GetPage(path string) *Page {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if page, ok := s.Pages[path]; ok {
		return page
	}

	return &Page{}
}

func (s *Statistics) GetVisitor(ip string) *Visitor {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if visitor, ok := s.Visitors[ip]; ok {
		return visitor
	}

	return &Visitor{}
}

func (s *Statistics) GetVisit(id int) *Visit {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if visit, ok := s.Visits[id]; ok {
		return visit
	}

	return &Visit{}
}

func (s *Statistics) VisitsCount() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return len(s.Visits)
}

func (s *Statistics) VisitorsCount() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return len(s.Visitors)
}

func (s *Statistics) EstimatedCurrentVisitors() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	estimatedCurrentVisitors := 0

	for _, v := range s.Visitors {
		if time.Since(v.LastDynamicVisit().Date) < v.AverageTimeSpent() {
			estimatedCurrentVisitors++
		}
	}

	return estimatedCurrentVisitors
}

func (s *Statistics) AverageDynamicVisitsPerVisitor() int {
	visitors := len(s.Visitors)
	totalVisits := 0

	for _, v := range s.Visitors {
		totalVisits += len(v.History)
	}

	return totalVisits / visitors
}

func (s *Statistics) MostVisitedPages() []*Page {
	s.mutex.Lock()

	pagesSlice := make([]*Page, len(s.Pages))

	for _, page := range s.Pages {
		pagesSlice = append(pagesSlice, page)
	}

	s.mutex.Unlock()

	slices.SortFunc(pagesSlice, func(a, b *Page) int {
		return cmp.Compare(len(a.Visits), len(b.Visits))
	})

	return pagesSlice
}

func (s *Statistics) LeastVisitedPages() []*Page {
	pagesSlice := s.MostVisitedPages()

	slices.Reverse(pagesSlice)

	return pagesSlice
}

func (s *Statistics) LanguagesCount() map[string]int {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.VisitorsLanguage
}

func (p *Page) VisitsCount() int {
	return len(p.Visits)
}

func (p *Page) VisitorsCount() int {
	visitors := make(map[*Visitor]bool)

	for _, v := range p.Visits {
		visitors[v.VisitedBy] = true
	}

	return len(visitors)
}

func (p *Page) AverageTimeSpent() time.Duration {
	i := 0
	totalTimeSpent := time.Duration(0)

	for _, v := range p.Visits {
		if v.Type == "route" {
			i++
			totalTimeSpent += v.TimeSpent
		}
	}

	return totalTimeSpent / time.Duration(i)
}

func (p *Page) AverageLoadingTime() time.Duration {
	i := 0
	totalLoadingTime := time.Duration(0)

	for _, v := range p.Visits {
		if v.Type == Dynamic {
			i++
			totalLoadingTime += v.LoadingTime
		}
	}

	return totalLoadingTime / time.Duration(i)
}

func (p *Page) GetVisit(date time.Time) (*Visit, error) {
	index, found := slices.BinarySearchFunc(p.Visits, &Visit{Date: date}, func(a, b *Visit) int {
		return a.Date.Compare(b.Date)
	})

	if found {
		return p.Visits[index], nil
	}

	return nil, fmt.Errorf("visit not found")
}

//func (v *Visitor) PrettyHistory() string {
//	history := ""
//
//	for p := range v.History {
//
//	}
//}

func (v *Visitor) VisitsCount() int {
	return len(v.History)
}

func (v *Visitor) AverageTimeSpent() time.Duration {
	i := 0
	totalTimeSpent := time.Duration(0)

	for _, vi := range v.History {
		if vi.Type == "route" {
			i++
			totalTimeSpent += vi.TimeSpent
		}
	}

	return totalTimeSpent / time.Duration(i)
}

func (v *Visitor) LastVisit() *Visit {
	return v.History[len(v.History)-1]
}

func (v *Visitor) LastDynamicVisit() *Visit {
	last := len(v.History)-1
	
	for i := range v.History {
		if v.History[last-i].Type == Dynamic {
			return v.History[last-i]
		}
	}

	return &Visit{}
}

func (v *Visitor) GetVisit(date time.Time) (*Visit, error) {
	index, found := slices.BinarySearchFunc(v.History, &Visit{Date: date}, func(a, b *Visit) int {
		return a.Date.Compare(b.Date)
	})

	if found {
		return v.History[index], nil
	}

	return nil, fmt.Errorf("visit not found")
}