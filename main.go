package main

import (
    "time"
    "log"
    "os"
    "github.com/gofiber/fiber/v2"
    "gorm.io/driver/mysql"
    "gorm.io/gorm"
    "gorm.io/gorm/logger"
    "github.com/gofiber/fiber/v2/middleware/cors"
    "github.com/gofiber/fiber/v2/middleware/limiter"
)

type Worker struct {
    ID          *int       `json:"id" gorm:"primaryKey"`
    Name        *string    `json:"name" gorm:"index"`
    Position    *string    `json:"position"`

    HatID       *string    `json:"hat_id"`
    HatStatus   *bool      `json:"hat_status" gorm:"-"` // Don't save to DB

    ShirtID     *string    `json:"shirt_id"`
    ShirtStatus *bool      `json:"shirt_status" gorm:"-"` // Don't save to DB

    BootID      *string    `json:"boot_id"`
    BootStatus  *bool      `json:"boot_status" gorm:"-"` // Don't save to DB

    GloveID     *string    `json:"glove_id"`
    GloveStatus *bool      `json:"glove_status" gorm:"-"` // Don't save to DB

    CreatedAt   *time.Time `json:"created_at"`
    LastestScan *time.Time `json:"lastest_scan" gorm:"-"` // Don't save to DB, calculate from scans
}

type Scan struct {
    ID           *int       `json:"id" gorm:"primaryKey"`
    WorkerID     *int       `json:"worker_id" gorm:"index"`
    ScanDate     *time.Time `json:"scan_date" gorm:"index"`
    ScanTime     *time.Time `json:"scan_time"`
    ScannedNFCID *string    `json:"scanned_nfc_id"`
    Status       *string    `json:"status"`
    Equipment    *string    `json:"equip_name"`
}

var DB *gorm.DB

func main() {
    // Configure GORM logger to suppress or reduce logs
    gormLogger := logger.Default.LogMode(logger.Error) // Only log errors

    // Connect DB with optimized settings
    dsn := "root:zTuGFSJnzSDtQQexCsJnakBWFIHUhCbH@tcp(shortline.proxy.rlwy.net:11710)/railway?charset=utf8mb4&parseTime=True&loc=Local"
    db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
        Logger: gormLogger,
        DisableAutomaticPing: false,
        PrepareStmt: true,
    })
    if err != nil {
        log.Fatal("DB connection failed:", err)
    }
    DB = db

    // Configure connection pool
    sqlDB, err := db.DB()
    if err != nil {
        log.Fatal("Failed to get database instance:", err)
    }
    sqlDB.SetMaxIdleConns(10)
    sqlDB.SetMaxOpenConns(100)
    sqlDB.SetConnMaxLifetime(time.Hour)

    // Only run migrations if explicitly enabled
    if os.Getenv("MIGRATE") == "true" {
        log.Println("Running database migrations...")
        if err := DB.AutoMigrate(&Worker{}, &Scan{}); err != nil {
            log.Fatal("Migration failed:", err)
        }
        log.Println("Migrations completed successfully")
    } else {
        log.Println("Skipping migrations (set MIGRATE=true to run)")
    }

    // Fiber app
    app := fiber.New(fiber.Config{
        DisableStartupMessage: false,
        ServerHeader:          "",
        AppName:               "Worker Management API",
    })

    app.Use(cors.New(cors.Config{
        AllowOrigins: "*",
        AllowMethods: "GET,POST,PUT,DELETE",
        AllowHeaders: "*",
    }))
    
    app.Use(limiter.New(limiter.Config{
        Max:        1,
        Expiration: 4 * time.Second,
    }))

    // Health check endpoint
    app.Get("/health", func(c *fiber.Ctx) error {
        return c.JSON(fiber.Map{
            "status": "ok",
            "time":   time.Now(),
        })
    })

    app.Get("/get_today_worker", func(c *fiber.Ctx) error {
        var workers []Worker
    
        loc, _ := time.LoadLocation("Asia/Bangkok")
        now := time.Now().In(loc)
        todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
        todayEnd := todayStart.Add(24 * time.Hour)
    
        // Get all workers
        if err := DB.Find(&workers).Error; err != nil {
            return c.Status(500).JSON(fiber.Map{"error": err.Error()})
        }
    
        // Get today's scans
        var scans []Scan
        err := DB.Where("scan_date >= ? AND scan_date < ?", todayStart, todayEnd).
            Find(&scans).Error
        if err != nil {
            return c.Status(500).JSON(fiber.Map{"error": err.Error()})
        }
    
        // Create worker map
        workerMap := make(map[int]*Worker, len(workers))
        f := false
        t := true
        
        for i := range workers {
            if workers[i].ID != nil {
                // Initialize all statuses to false
                workers[i].HatStatus = &f
                workers[i].ShirtStatus = &f
                workers[i].BootStatus = &f
                workers[i].GloveStatus = &f
                workers[i].LastestScan = nil
                workerMap[*workers[i].ID] = &workers[i]
            }
        }
    
        // Process scans and check if NFC ID matches worker's equipment
        for _, scan := range scans {
            if scan.WorkerID == nil || scan.ScannedNFCID == nil || scan.ScanTime == nil {
                continue
            }
            
            worker := workerMap[*scan.WorkerID]
            if worker == nil {
                continue
            }
    
            // Check if scanned NFC matches worker's equipment IDs
            if worker.HatID != nil && *scan.ScannedNFCID == *worker.HatID {
                worker.HatStatus = &t
            }
            
            if worker.ShirtID != nil && *scan.ScannedNFCID == *worker.ShirtID {
                worker.ShirtStatus = &t
            }
            
            if worker.BootID != nil && *scan.ScannedNFCID == *worker.BootID {
                worker.BootStatus = &t
            }
            
            if worker.GloveID != nil && *scan.ScannedNFCID == *worker.GloveID {
                worker.GloveStatus = &t
            }
    
            // Update latest scan time
            if worker.LastestScan == nil || scan.ScanTime.After(*worker.LastestScan) {
                worker.LastestScan = scan.ScanTime
            }
        }
    
        return c.JSON(workers)
    })

    app.Get("/get_all_worker", func(c *fiber.Ctx) error {
        var workers []Worker

        err := DB.Find(&workers).Error
        if err != nil {
            return c.Status(500).JSON(fiber.Map{"error": err.Error()})
        }

        return c.JSON(workers)
    })

    app.Get("/get_scan", func(c *fiber.Ctx) error {
        var scans []Scan

        loc, _ := time.LoadLocation("Asia/Bangkok")
        today := time.Now().In(loc).Format("2006-01-02")

        err := DB.Where("DATE(scan_date) = ?", today).Find(&scans).Error
        if err != nil {
            return c.Status(500).JSON(fiber.Map{"error": err.Error()})
        }

        return c.JSON(scans)
    })

    port := os.Getenv("PORT")
    if port == "" {
        port = "3000"
    }

    log.Println("Server starting on port:", port)
    log.Fatal(app.Listen(":" + port))
}