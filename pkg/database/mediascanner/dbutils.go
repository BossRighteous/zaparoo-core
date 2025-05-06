package mediascanner

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ZaparooProject/zaparoo-core/pkg/database"
	"github.com/ZaparooProject/zaparoo-core/pkg/utils"
)

// We can't batch effectively without a sense of relationships
// Instead of indexing string columns use in-mem map to track records to
// insert IDs. Batching should be able to run with assumed IDs

func AddMediaPath(db database.MediaDBI, systemDBID int64, path string) error {
	pf := GetPathFragments(path)

	title, err := db.FindOrInsertMediaTitle(database.MediaTitle{
		Slug:       pf.Slug,
		Name:       pf.Title,
		SystemDBID: systemDBID,
	})
	if err != nil {
		fmt.Println("Error: FindOrInsertMediaTitle")
		return err
	}

	media, err := db.FindMedia(database.Media{
		Path:           pf.Path,
		MediaTitleDBID: title.DBID,
	})
	if err != nil && err != sql.ErrNoRows {
		fmt.Println("Error: FindMedia")
		return err
	}
	if err == nil && media.DBID > 0 {
		fmt.Println("Media Found! Setting to Active")
		tsql := db.UnsafeGetSqlDB()
		tsql.Exec(`update Media set IsActive = 1 where DBID = ?`, media.DBID)
		return nil
	}

	fmt.Println("Adding media", pf.Path)
	media, err = db.InsertMedia(database.Media{
		Path:           pf.Path,
		MediaTitleDBID: title.DBID,
	})
	if err != nil {
		fmt.Println("Error: InsertMedia")
		return err
	}

	if pf.Ext != "" {
		tag, err := db.FindOrInsertTag(database.Tag{
			Tag:      pf.Ext,
			TypeDBID: int64(2),
		})
		if err != nil {
			fmt.Println("Error: FindOrInsertTag Ext")
			return err
		}

		_, err = db.FindOrInsertMediaTag(database.MediaTag{
			TagDBID:   tag.DBID,
			MediaDBID: media.DBID,
		})
		if err != nil {
			fmt.Println("Error: FindOrInsertMediaTag Ext")
			return err
		}

	}

	for _, tagStr := range pf.Tags {
		tag, err := db.FindTag(database.Tag{
			Tag: tagStr,
		})
		if err != nil && err != sql.ErrNoRows {
			fmt.Println("Error: FindTag", err)
			return err
		}
		if tag.DBID > 0 {
			_, err = db.FindOrInsertMediaTag(database.MediaTag{
				TagDBID:   tag.DBID,
				MediaDBID: media.DBID,
			})
			if err != nil {
				fmt.Println("Error: FindOrInsertMediaTag Ext")
				return err
			}
		}
	}
	return nil
}

type MediaPathFragments struct {
	Path     string
	FileName string
	Title    string
	Slug     string
	Ext      string
	Tags     []string
}

func getTagsFromFileName(filename string) []string {
	re := regexp.MustCompile(`\(([\w\,\- ]*)\)|\[([\w\,\- ]*)\]`)
	matches := re.FindAllString(filename, -1)
	tags := make([]string, 0)
	for _, padded := range matches {
		unpadded := padded[1 : len(padded)-1]
		split := strings.Split(unpadded, ",")
		for _, tag := range split {
			tags = append(tags, strings.ToLower(strings.TrimSpace(tag)))
		}
	}
	return tags
}

func getTitleFromFilename(filename string) string {
	r := regexp.MustCompile(`^([^\(\[]*)`)
	title := r.FindString(filename)
	return strings.TrimSpace(title)
}

func SeedKnownTags(db database.MediaDBI) error {
	typeMatches := map[string][]string{
		//"Unknown": {"unknown"},
		//"Extension": {".ext"},
		"Version": {
			"rev", "v",
		},
		"Language": {
			"ar", "bg", "bs", "cs", "cy", "da", "de", "el", "en", "eo", "es", "et",
			"fa", "fi", "fr", "ga", "gu", "he", "hi", "hr", "hu", "is", "it", "ja",
			"ko", "lt", "lv", "ms", "nl", "no", "pl", "pt", "ro", "ru", "sk", "sl",
			"sq", "sr", "sv", "th", "tr", "ur", "vi", "yi", "zh",
		},
		"Region": {
			// NOINTRO
			"world", "europe", "asia", "australia", "brazil", "canada", "china", "france",
			"germany", "hong kong", "italy", "japan", "korea", "netherlands", "spain",
			"sweden", "usa", "poland", "finland", "denmark", "portugal", "norway",
			// TOSEC
			"AE", "AL", "AS", "AT", "AU", "BA", "BE", "BG", "BR", "CA", "CH", "CL", "CN",
			"CS", "CY", "CZ", "DE", "DK", "EE", "EG", "ES", "EU", "FI", "FR", "GB", "GR",
			"HK", "HR", "HU", "ID", "IE", "IL", "IN", "IR", "IS", "IT", "JO", "JP", "KR",
			"LT", "LU", "LV", "MN", "MX", "MY", "NL", "NO", "NP", "NZ", "OM", "PE", "PH",
			"PL", "PT", "QA", "RO", "RU", "SE", "SG", "SI", "SK", "TH", "TR", "TW", "US",
			"VN", "YU", "ZA",
		},
		"Year": {
			"1970", "1971", "1972", "1973", "1974", "1975", "1976", "1977", "1978", "1979",
			"1980", "1981", "1982", "1983", "1984", "1985", "1986", "1987", "1988", "1989",
			"1990", "1991", "1992", "1993", "1994", "1995", "1996", "1997", "1998", "1999",
			"2000", "2001", "2002", "2003", "2004", "2005", "2006", "2007", "2008", "2009",
			"2010", "2011", "2012", "2013", "2014", "2015", "2016", "2017", "2018", "2019",
			"2020", "2021", "2022", "2023", "2024", "2025", "2026", "2027", "2028", "2029",
			"19xx", "197x", "198x", "199x", "20xx", "200x", "201x", "202x",
		},
		"Dev Status": {
			"alpha", "beta", "preview", "pre-release", "proto", "sample",
			"demo", "demo-kiosk", "demo-playable", "demo-rolling", "demo-slideshow",
		},
		"Media Type": {
			"disc", "disk", "file", "part", "side", "tape",
		},
		"TOSEC System": {
			"+2", "+2a", "+3", "130XE", "A1000", "A1200", "A1200-A4000", "A2000",
			"A2000-A3000", "A2024", "A2500-A3000UX", "A3000", "A4000", "A4000T",
			"A500", "A500+", "A500-A1000-A2000", "A500-A1000-A2000-CDTV", "A500-A1200",
			"A500-A1200-A2000-A4000", "A500-A2000", "A500-A600-A2000", "A570", "A600",
			"A600HD", "AGA", "AGA-CD32", "Aladdin Deck Enhancer", "CD32", "CDTV",
			"Computrainer", "Doctor PC Jr.", "ECS", "ECS-AGA", "Executive", "Mega ST",
			"Mega-STE", "OCS", "OCS-AGA", "ORCH80", "Osbourne 1", "PIANO90",
			"PlayChoice-10", "Plus4", "Primo-A", "Primo-A64", "Primo-B", "Primo-B64",
			"Pro-Primo", "ST", "STE", "STE-Falcon", "TT", "TURBO-R GT", "TURBO-R ST",
			"VS DualSystem", "VS UniSystem",
		},
		"TOSEC Video": {
			"CGA", "EGA", "HGC", "MCGA", "MDA", "NTSC", "NTSC-PAL", "PAL", "PAL-60",
			"PAL-NTSC", "SVGA", "VGA", "XGA",
		},
		"TOSEC Copyright": {
			"CW", "CW-R", "FW", "GW", "GW-R", "LW", "PD", "SW", "SW-R",
		},
		"TOSEC Dump Info": {
			"cr", "f", "h", "m", "p", "t", "tr", "o", "u", "v", "b", "a", "!",
		},
	}

	_, err := db.InsertTagType(database.TagType{
		DBID: int64(1),
		Type: "Unknown",
	})
	if err != nil {
		return err
	}
	_, err = db.InsertTag(database.Tag{
		Tag:      "unknown",
		TypeDBID: int64(1),
	})
	if err != nil {
		return err
	}

	_, err = db.InsertTagType(database.TagType{
		DBID: int64(2),
		Type: "Extension",
	})
	if err != nil {
		return err
	}
	_, err = db.InsertTag(database.Tag{
		Tag:      ".ext",
		TypeDBID: int64(2),
	})
	if err != nil {
		return err
	}

	for typeStr, tags := range typeMatches {
		tagType, err := db.InsertTagType(database.TagType{
			Type: typeStr,
		})
		if err != nil {
			return err
		}

		for _, tag := range tags {
			_, err := db.InsertTag(database.Tag{
				Tag:      strings.ToLower(tag),
				TypeDBID: tagType.DBID,
			})
			if err != nil {
				return err
			}

		}
	}
	return nil
}

func GetPathFragments(path string) MediaPathFragments {
	f := MediaPathFragments{}
	f.Path = filepath.Clean(path)
	fileBase := filepath.Base(f.Path)
	f.Ext = filepath.Ext(f.Path)
	f.FileName, _ = strings.CutSuffix(fileBase, f.Ext)
	f.Title = getTitleFromFilename(f.FileName)
	f.Slug = utils.SlugifyString(f.Title)
	f.Tags = getTagsFromFileName(f.FileName)
	return f
}
