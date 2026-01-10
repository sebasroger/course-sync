package main

import (
	"context"
	"flag"
	"log"
	"path/filepath"
	"time"

	"course-sync/internal/config"
	"course-sync/internal/export"
	"course-sync/internal/providers/pluralsight"
	"course-sync/internal/providers/udemy"
	"course-sync/internal/sftpclient"
)

func main() {
	var (
		outPath    = flag.String("out", "COURSE-MAIN_ALL.csv", "output csv path")
		udemyPages = flag.Int("udemy-max-pages", 1, "max pages to fetch from udemy")
		psPages    = flag.Int("ps-max-pages", 1, "max pages to fetch from pluralsight")
		pageSize   = flag.Int("page-size", 100, "page size for providers (when supported)")

		uploadSFTP = flag.Bool("sftp", false, "upload the generated CSV via SFTP")
	)
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	cfg := config.Load()

	u := udemy.New(cfg.UdemyBaseURL, cfg.UdemyClientID, cfg.UdemyClientSecret)
	p := pluralsight.New(cfg.PluralsightBaseURL, cfg.PluralsightToken)

	udProv := udemy.Provider{
		C:        u,
		PageSize: *pageSize,
		MaxPages: *udemyPages,
	}
	udCourses, err := udProv.ListCourses(ctx)
	if err != nil {
		log.Fatal(err)
	}

	psProv := pluralsight.Provider{
		C:        p,
		First:    *pageSize,
		MaxPages: *psPages,
	}
	psCourses, err := psProv.ListCourses(ctx)
	if err != nil {
		log.Fatal(err)
	}

	all := append(udCourses, psCourses...)
	if err := export.WriteEightfoldCourseCSV(*outPath, all); err != nil {
		log.Fatal(err)
	}
	log.Printf("wrote %d courses to %s", len(all), *outPath)

	if *uploadSFTP {
		remoteName := filepath.Base(*outPath)

		upCfg := sftpclient.Config{
			Host:                  cfg.SFTPHost,
			Port:                  cfg.SFTPPort,
			User:                  cfg.SFTPUser,
			Pass:                  cfg.SFTPPass,
			RemoteDir:             cfg.SFTPDir,
			InsecureIgnoreHostKey: cfg.SFTPInsecureIgnoreHostKey,
		}

		upCtx, upCancel := context.WithTimeout(ctx, 2*time.Minute)
		defer upCancel()

		if err := sftpclient.UploadFile(upCtx, upCfg, *outPath, remoteName); err != nil {
			log.Fatal(err)
		}
		log.Printf("uploaded to sftp://%s:%d%s/%s", upCfg.Host, upCfg.Port, upCfg.RemoteDir, remoteName)
	}
}
