Show stats about the kai brain.

Run the following and summarize the results:

1. Count brain files by category:
```bash
find brain -name "*.md" | grep -v "^brain/index" | awk -F/ '{print $2}' | sort | uniq -c | sort -rn
```

2. Total brain files and size:
```bash
find brain -name "*.md" | wc -l
du -sh brain/
```

3. Check index freshness (compare index.json mtime vs newest brain file):
```bash
ls -la brain/index.json brain/vectors.bin.gz 2>/dev/null
find brain -name "*.md" -newer brain/index.json 2>/dev/null | head -10
```

4. List the 5 most recently modified brain files:
```bash
find brain -name "*.md" -exec stat -f "%m %N" {} \; | sort -rn | head -5 | awk '{print $2}'
```

Summarize: total files, breakdown by category, whether the index needs rebuilding (any .md files newer than index.json), and the most recently updated topics.
