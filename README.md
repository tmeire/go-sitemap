# go-sitemap

Current limitations of the optimised parser:
- Only works for buffers that can contain the whole urlset
- Does not work for <sitemaps>
- Does not work when document includes <?xml tags
- Does not work when element has attributes