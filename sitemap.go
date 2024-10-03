package gositemap

import (
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"strconv"
	"time"
)

// https://www.sitemaps.org/protocol.html

type Frequency int

const (
	UKNOWN Frequency = iota
	ALWAYS
	HOURLY
	DAILY
	WEEKLY
	MONTHLY
	YEARLY
	NEVER
)

func (f Frequency) String() string {
	switch f {
	case ALWAYS:
		return "always"
	case HOURLY:
		return "hourly"
	case DAILY:
		return "daily"
	case WEEKLY:
		return "weekly"
	case MONTHLY:
		return "monthly"
	case YEARLY:
		return "yearly"
	case NEVER:
		return "never"
	}
	return "unknown"
}

func (f *Frequency) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var v string
	err := d.DecodeElement(&v, &start)
	if err != nil {
		return err
	}

	switch v {
	case "always":
		*f = ALWAYS
	case "hourly":
		*f = HOURLY
	case "daily":
		*f = DAILY
	case "weekly":
		*f = WEEKLY
	case "monthly":
		*f = MONTHLY
	case "yearly":
		*f = YEARLY
	case "never":
		*f = NEVER
	default:
		*f = UKNOWN
	}
	return nil
}

type TimeISO3339 struct {
	time.Time
}

const formatISO3339NoMinutes = "2006-01-02T15:04Z"

func (t *TimeISO3339) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var v string
	err := d.DecodeElement(&v, &start)
	if err != nil {
		return err
	}

	tt, err := time.Parse(formatISO3339NoMinutes, v)
	if err != nil {
		return err
	}

	t.Time = tt
	return nil
}

type BoundedFloat64 float64

func (bf *BoundedFloat64) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var fs string
	err := d.DecodeElement(&fs, &start)
	if err != nil {
		return err
	}

	f, err := strconv.ParseFloat(fs, 64)
	if err != nil {
		f = .5
	}

	if f < 0. {
		f = 0.
	}
	if f > 1. {
		f = 1.
	}
	if math.IsNaN(f) {
		f = .5
	}

	*bf = BoundedFloat64(f)

	return nil
}

type URL struct {
	XMLName    xml.Name        `xml:"url"`
	Loc        string          `xml:"loc"`
	Lastmod    TimeISO3339     `xml:"lastmod"`
	Changefreq Frequency       `xml:"changefreq"`
	Priority   *BoundedFloat64 `xml:"priority"`
}

type URLSet struct {
	XMLName xml.Name `xml:"urlset"`
	Urls    []URL    `xml:"url"`
}

type SiteMaps struct {
	XMLName xml.Name  `xml:"sitemapindex"`
	Maps    []SiteMap `xml:"sitemap"`
}

type SiteMap struct {
	XMLName xml.Name    `xml:"sitemap"`
	Loc     string      `xml:"loc"`
	Lastmod TimeISO3339 `xml:"lastmod"`
}

type SiteMapOrURLSet struct {
	Maps []SiteMap
	URLs []URL
}

func (s *SiteMapOrURLSet) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	switch start.Name.Local {
	case "sitemapindex":
		var sitemaps SiteMaps
		err := d.DecodeElement(&sitemaps, &start)
		if err != nil {
			return nil
		}
		s.Maps = sitemaps.Maps
		return nil
	case "urlset":
		var urlset URLSet
		err := d.DecodeElement(&urlset, &start)
		if err != nil {
			return err
		}
		s.URLs = urlset.Urls
		return nil
	default:
		return fmt.Errorf("unexpected tag %s", start.Name.Local)
	}
}

func ParseReaderNative(content io.Reader) (*SiteMapOrURLSet, error) {
	var u SiteMapOrURLSet
	err := xml.NewDecoder(content).Decode(&u)
	if err != nil {
		return nil, err
	}

	return &u, nil
}

func ParseReaderOptimized(content io.Reader) (*SiteMapOrURLSet, error) {
	bs := make([]byte, 64)
	n, err := content.Read(bs)
	for err == nil {
		//fmt.Printf("%q, %d, %s\n", string(bs[:n]), n, err)
		for i := 0; i < n; i++ {
			switch bs[i] {
			case '\n', '\t', ' ':
				continue
			//case '<':
			//	fmt.Println(bs[i])
			default:
				fmt.Print(string(bs[i]))
			}
		}
		n, err = content.Read(bs)
		//fmt.Printf("%q, %d, %s", string(bs[:n]), n, err)
	}

	if err != io.EOF {
		return nil, err
	}

	var res SiteMapOrURLSet
	return &res, nil
}
