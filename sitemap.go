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

type parseLevel int

const (
	root parseLevel = iota
	urlset
	url
	loc
	lastmod
	priority
	changefreq
)

func ParseReaderOptimized(content io.Reader) (*SiteMapOrURLSet, error) {
	bs := make([]byte, 512)
	n, err := content.Read(bs)
	currentParseLevel := root
	contentStart := -1
	var currentURLSet URLSet
	var currentURL *URL
	for err == nil {
		//fmt.Printf("%q, %d, %s\n", string(bs[:n]), n, err)
		for i := 0; i < n; i++ {
			switch bs[i] {
			case '\n', '\t', ' ':
				continue
			case '<':
				// [/] urlset, url, loc, lastmod, priority, changefreq
				switch currentParseLevel {
				case root:
					if string(bs[i+1:i+8]) != "urlset>" {
						return nil, fmt.Errorf("unexpected tag at position %d", i+1)
					}
					currentURLSet = URLSet{}
					currentParseLevel = urlset
					i += 7
				case urlset:
					if string(bs[i+1:i+5]) == "url>" {
						currentURL = &URL{}
						currentParseLevel = url
						i += 4
					} else if string(bs[i+1:i+9]) == "/urlset>" {
						currentParseLevel = root
						i += 8
					} else {
						return nil, fmt.Errorf("unexpected tag at position %d", i+1)
					}
				case url:
					switch bs[i+2] {
					case 'o': // loc
						if string(bs[i+1:i+5]) != "loc>" {
							return nil, fmt.Errorf("unexpected tag at position %d, expected 'loc'", i+1)
						}
						contentStart = i + 5
						currentParseLevel = loc
						i += 4
					case 'a': // lastmod
						if string(bs[i+1:i+9]) != "lastmod>" {
							return nil, fmt.Errorf("unexpected tag at position %d, expected 'lastmod'", i+1)
						}
						contentStart = i + 9
						currentParseLevel = lastmod
						i += 8
					case 'r': // priority
						if string(bs[i+1:i+10]) != "priority>" {
							return nil, fmt.Errorf("unexpected tag at position %d, expected 'priority'", i+1)
						}
						contentStart = i + 10
						currentParseLevel = priority
						i += 9
					case 'h': // changefreq
						if string(bs[i+1:i+12]) != "changefreq>" {
							return nil, fmt.Errorf("unexpected tag at position %d, expected 'changefreq'", i+1)
						}
						contentStart = i + 12
						currentParseLevel = changefreq
						i += 11
					case 'u': // close the url element
						if string(bs[i+1:i+6]) != "/url>" {
							return nil, fmt.Errorf("unexpected tag at position %d, expected 'changefreq'", i+1)
						}
						currentURLSet.Urls = append(currentURLSet.Urls, *currentURL)
						currentURL = nil
						currentParseLevel = urlset
						i += 5
					default:
						return nil, fmt.Errorf("unexpected tag at position %d", i+1)
					}
				case loc:
					if string(bs[i+1:i+6]) != "/loc>" {
						return nil, fmt.Errorf("unexpected tag at position %d, expected '</loc>'", i+1)
					}
					currentURL.Loc = string(bs[contentStart:i])
					currentParseLevel = url
					i += 5
				case changefreq:
					if string(bs[i+1:i+13]) != "/changefreq>" {
						return nil, fmt.Errorf("unexpected tag at position %d, expected '</changefreq>'", i+1)
					}

					switch string(bs[contentStart:i]) {
					case "always":
						currentURL.Changefreq = ALWAYS
					case "hourly":
						currentURL.Changefreq = HOURLY
					case "daily":
						currentURL.Changefreq = DAILY
					case "weekly":
						currentURL.Changefreq = WEEKLY
					case "monthly":
						currentURL.Changefreq = MONTHLY
					case "yearly":
						currentURL.Changefreq = YEARLY
					case "never":
						currentURL.Changefreq = NEVER
					default:
						currentURL.Changefreq = UKNOWN
					}
					currentParseLevel = url
					i += 12
				case priority:
					if string(bs[i+1:i+11]) != "/priority>" {
						return nil, fmt.Errorf("unexpected tag at position %d, expected '</priority>'", i+1)
					}
					f, err := strconv.ParseFloat(string(bs[contentStart:i]), 64)
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
					currentURL.Priority = (*BoundedFloat64)(&f)
					currentParseLevel = url
					i += 10
				case lastmod:
					if string(bs[i+1:i+10]) != "/lastmod>" {
						return nil, fmt.Errorf("unexpected tag at position %d, expected '</lastmod>'", i+1)
					}

					tt, err := time.Parse(formatISO3339NoMinutes, string(bs[contentStart:i]))
					if err != nil {
						return nil, fmt.Errorf("unexpected value %s for lastmod at position %d", string(bs[contentStart:i]), contentStart)
					}

					currentURL.Lastmod.Time = tt
					currentParseLevel = url
					i += 9
				}
			default:
				switch currentParseLevel {
				case root, urlset, url:
					return nil, fmt.Errorf("unexpected character at position %d", i)
				}
				//fmt.Print(string(bs[i]))
			}
		}
		n, err = content.Read(bs)
		//fmt.Printf("%q, %d, %s", string(bs[:n]), n, err)
	}

	if err != io.EOF {
		return nil, err
	}

	return &SiteMapOrURLSet{
		URLs: currentURLSet.Urls,
	}, nil
}
