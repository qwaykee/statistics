package statistics

import (
	"github.com/gin-gonic/gin"

	"time"
	"sync"
	"slices"
	"cmp"
	"fmt"
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
		History []*Visit
	}

	Visit struct {
		ID int
		Date time.Time
		LoadingTime time.Duration
		TimeSpent time.Duration
		CodeIssued int
		Referer string
		VisitedBy *Visitor
		Page *Page
	}

	pagesSlice []*Page
)

func New() *Statistics {
	return &Statistics{
		Pages: make(map[string]*Page),
		Visitors: make(map[string]*Visitor),
		Visits: make(map[int]*Visit),
		VisitorsLanguage: make(map[string]int),
	}
}

func (s *Statistics) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
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

			s.VisitorsLanguage[lang] = s.VisitorsLanguage[lang] + 1
		}

		visitor := s.Visitors[visitorIP]
		page := s.Pages[pagePath]

		if len(visitor.History) > 1 {
			visitor.LastVisit().TimeSpent = time.Since(visitor.LastVisit().Date)
		}

		s.currentVisitID++

		visit := &Visit{
			ID: s.currentVisitID,
			Date: time.Now(),
			TimeSpent: 0,
			Referer: c.GetHeader("Referer"),
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

	visits := 0

	for _, p := range s.Pages {
		visits += len(p.Visits)
	}

	return visits
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
		if time.Since(v.LastVisit().Date) < v.AverageTimeSpent() {
			estimatedCurrentVisitors++
		}
	}

	return estimatedCurrentVisitors
}

func (s *Statistics) AverageVisitsPerVisitor() int {
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
	visits := len(p.Visits)
	totalTimeSpent := time.Duration(0)

	for _, v := range p.Visits {
		totalTimeSpent += v.TimeSpent
	}

	return totalTimeSpent / time.Duration(visits)
}

func (p *Page) AverageLoadingTime() time.Duration {
	visits := len(p.Visits)
	totalLoadingTime := time.Duration(0)

	for _, v := range p.Visits {
		totalLoadingTime += v.LoadingTime
	}

	return totalLoadingTime / time.Duration(visits)
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
	visits := len(v.History)
	totalTimeSpent := time.Duration(0)

	for _, vi := range v.History {
		totalTimeSpent += vi.TimeSpent
	}

	return totalTimeSpent / time.Duration(visits)
}

func (v *Visitor) LastVisit() *Visit {
	return v.History[len(v.History)-1]
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