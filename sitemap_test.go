package gositemap_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	gositemap "github.com/tmeire/go-sitemap"
)

func TestParseSiteMapOrURLSet(t *testing.T) {
	tests := []struct {
		file   string
		verify func(t *testing.T, s gositemap.SiteMapOrURLSet)
	}{
		// {
		// 	"./testdata/vtm-sitemap.xml",
		// 	func(t *testing.T, s gositemap.SiteMapOrURLSet) {
		// 		assert.NotNil(t, s.Maps)
		// 		assert.Nil(t, s.URLs)

		// 		assert.Len(t, s.Maps, 2)

		// 		assert.Equal(t, "https://koken.vtm.be/sitemap.xml?page=1", s.Maps[0].Loc)
		// 		assert.Equal(t, "2024-02-23T08:20:00Z", s.Maps[0].Lastmod.Format(time.RFC3339))

		// 		assert.Equal(t, "https://koken.vtm.be/sitemap.xml?page=2", s.Maps[1].Loc)
		// 		assert.Equal(t, "2024-02-23T08:20:00Z", s.Maps[1].Lastmod.Format(time.RFC3339))
		// 	},
		// },
		{
			"./testdata/vtm-page-1.xml",
			func(t *testing.T, s gositemap.SiteMapOrURLSet) {
				assert.Nil(t, s.Maps)
				assert.NotNil(t, s.URLs)

				assert.Len(t, s.URLs, 10000)

				assert.Equal(t, "https://koken.vtm.be/", s.URLs[0].Loc)
				assert.Equal(t, gositemap.DAILY, s.URLs[0].Changefreq)
				assert.Equal(t, gositemap.BoundedFloat64(1.0), *(s.URLs[0].Priority))

				assert.Equal(t, "https://koken.vtm.be/groenten-en-vis-van-gio/recept/toast-champignon-new-style", s.URLs[1].Loc)
				assert.Equal(t, gositemap.NEVER, s.URLs[1].Changefreq)
				assert.Equal(t, gositemap.BoundedFloat64(0.8), *s.URLs[1].Priority)
				assert.Equal(t, "2013-10-29T16:40:00Z", s.URLs[1].Lastmod.Format(time.RFC3339))
			},
		},
		{
			"./testdata/vtm-page-2.xml",
			func(t *testing.T, s gositemap.SiteMapOrURLSet) {
				assert.Nil(t, s.Maps)
				assert.NotNil(t, s.URLs)

				assert.Len(t, s.URLs, 657)

				var prio *gositemap.BoundedFloat64
				assert.Equal(t, "https://koken.vtm.be/kooktips/hoe-klop-ik-luchtige-boter", s.URLs[0].Loc)
				assert.Equal(t, gositemap.NEVER, s.URLs[0].Changefreq)
				assert.Equal(t, prio, s.URLs[0].Priority)
				assert.Equal(t, "2014-04-08T09:34:00Z", s.URLs[0].Lastmod.Format(time.RFC3339))
			},
		},
	}
	parsers := map[string]parser{
		//"native":    gositemap.ParseReaderNative,
		"optimised": gositemap.ParseReaderOptimized,
	}
	for name, p := range parsers {
		for _, tt := range tests {
			t.Run(fmt.Sprintf("%s-%s", name, tt.file), func(t *testing.T) {
				f, err := os.Open(tt.file)
				assert.NoError(t, err)

				sm, err := p(f)
				assert.NoError(t, err)

				tt.verify(t, *sm)
			})
		}
	}
}

type parser func(io.Reader) (*gositemap.SiteMapOrURLSet, error)

func TestURL(t *testing.T) {
	tests := []struct {
		name        string
		content     io.Reader
		verifyError assert.ErrorAssertionFunc
		loc         string
		changefreq  gositemap.Frequency
		priority    gositemap.BoundedFloat64
		lastmod     string
	}{
		{
			name: "ok",
			content: bytes.NewBufferString(`
				<urlset><url><loc>https://koken.vtm.be/</loc><changefreq>daily</changefreq><priority>1.0</priority></url></urlset>
			`),
			verifyError: assert.NoError,
			loc:         "https://koken.vtm.be/",
			changefreq:  gositemap.DAILY,
			priority:    1.0,
			lastmod:     "0001-01-01T00:00:00Z",
		},
		{
			name: "unknown change frequency",
			content: bytes.NewBufferString(`
				<urlset><url><loc>https://koken.vtm.be/</loc><changefreq>foobar</changefreq><priority>1.0</priority></url></urlset>
			`),
			verifyError: assert.NoError,
			loc:         "https://koken.vtm.be/",
			changefreq:  gositemap.UKNOWN,
			priority:    1.0,
			lastmod:     "0001-01-01T00:00:00Z",
		},
		{
			name: "priority outside upper bound",
			content: bytes.NewBufferString(`
				<urlset><url><loc>https://koken.vtm.be/</loc><changefreq>daily</changefreq><priority>1.1</priority></url></urlset>
			`),
			verifyError: assert.NoError,
			loc:         "https://koken.vtm.be/",
			changefreq:  gositemap.DAILY,
			priority:    1.0,
			lastmod:     "0001-01-01T00:00:00Z",
		},
		{
			name: "priority outside lower bound",
			content: bytes.NewBufferString(`
				<urlset><url><loc>https://koken.vtm.be/</loc><changefreq>daily</changefreq><priority>-1.1</priority></url></urlset>
			`),
			verifyError: assert.NoError,
			loc:         "https://koken.vtm.be/",
			changefreq:  gositemap.DAILY,
			priority:    0.0,
			lastmod:     "0001-01-01T00:00:00Z",
		},
		{
			name: "priority outside lower bound",
			content: bytes.NewBufferString(`
				<urlset><url><loc>https://koken.vtm.be/</loc><changefreq>daily</changefreq><priority>NaN</priority></url></urlset>
			`),
			verifyError: assert.NoError,
			loc:         "https://koken.vtm.be/",
			changefreq:  gositemap.DAILY,
			priority:    0.5,
			lastmod:     "0001-01-01T00:00:00Z",
		},
		{
			name: "priority +infinity",
			content: bytes.NewBufferString(`
				<urlset><url><loc>https://koken.vtm.be/</loc><changefreq>daily</changefreq><priority>+inf</priority></url></urlset>
			`),
			verifyError: assert.NoError,
			loc:         "https://koken.vtm.be/",
			changefreq:  gositemap.DAILY,
			priority:    1.,
			lastmod:     "0001-01-01T00:00:00Z",
		},
		{
			name: "priority -infinity",
			content: bytes.NewBufferString(`
				<urlset><url><loc>https://koken.vtm.be/</loc><changefreq>daily</changefreq><priority>-inf</priority></url></urlset>
			`),
			verifyError: assert.NoError,
			loc:         "https://koken.vtm.be/",
			changefreq:  gositemap.DAILY,
			priority:    0.,
			lastmod:     "0001-01-01T00:00:00Z",
		},
		{
			name: "priority garbage",
			content: bytes.NewBufferString(`
				<urlset><url><loc>https://koken.vtm.be/</loc><changefreq>daily</changefreq><priority>foobar</priority></url></urlset>
			`),
			verifyError: assert.NoError,
			loc:         "https://koken.vtm.be/",
			changefreq:  gositemap.DAILY,
			priority:    .5,
			lastmod:     "0001-01-01T00:00:00Z",
		},
	}

	parsers := map[string]parser{
		//"native": gositemap.ParseReaderNative,
		"optimised": gositemap.ParseReaderOptimized,
	}
	for name, p := range parsers {
		for _, tt := range tests {
			t.Run(fmt.Sprintf("%s-%s", name, tt.name), func(t *testing.T) {
				u, err := p(tt.content)
				tt.verifyError(t, err)

				assert.NotNil(t, u.URLs)
				assert.Len(t, u.URLs, 1)
				assert.Equal(t, tt.loc, u.URLs[0].Loc)
				assert.Equal(t, tt.changefreq, u.URLs[0].Changefreq)
				assert.Equal(t, tt.priority, *(u.URLs[0].Priority))
				assert.Equal(t, tt.lastmod, u.URLs[0].Lastmod.Format(time.RFC3339))
			})
		}
	}
}

func BenchmarkFullFiles(b *testing.B) {
	files := []string{
		"./testdata/vtm-page-1.xml",
		"./testdata/vtm-page-2.xml",
	}
	for i := 0; i < b.N; i++ {
		for _, file := range files {
			f, err := os.Open(file)
			if err != nil {
				panic(err)
			}

			_, _ = gositemap.ParseReaderOptimized(f)
		}
	}
}
