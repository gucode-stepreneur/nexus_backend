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
    HatStatus   *bool      `json:"hat_status" gorm:"default:false"`

    ShirtID     *string    `json:"shirt_id"`
    ShirtStatus *bool      `json:"shirt_status" gorm:"default:false"`

    BootID      *string    `json:"boot_id"`
    BootStatus  *bool      `json:"boot_status" gorm:"default:false"`

    GloveID     *string    `json:"glove_id"`
    GloveStatus *bool      `json:"glove_status" gorm:"default:false"`

    CreatedAt   *time.Time `json:"created_at"`
    LastestScan *time.Time `json:"lastest_scan" gorm:"index"`
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
    
    // If you want to see warnings too, use:
    // gormLogger := logger.Default.LogMode(logger.Warn)

    // Connect DB with optimized settings
    dsn := "root:zTuGFSJnzSDtQQexCsJnakBWFIHUhCbH@tcp(shortline.proxy.rlwy.net:11710)/railway?charset=utf8mb4&parseTime=True&loc=Local"
    db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
        Logger: gormLogger,
        // Disable automatic ping on startup
        DisableAutomaticPing: false,
        // Prepare statements for better performance
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
    // Run with: MIGRATE=true go run main.go
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
        // Reduce overhead
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
    
        // Workers - optimize with Select to only fetch needed fields if possible
        if err := DB.Find(&workers).Error; err != nil {
            return c.Status(500).JSON(fiber.Map{"error": err.Error()})
        }
    
        // Scans with indexed query
        var scans []Scan
        err := DB.Where("scan_date >= ? AND scan_date < ?", todayStart, todayEnd).
            Find(&scans).Error
        if err != nil {
            return c.Status(500).JSON(fiber.Map{"error": err.Error()})
        }
    
        workerMap := make(map[int]*Worker, len(workers))
        for i := range workers {
            if workers[i].ID != nil {
                workerMap[*workers[i].ID] = &workers[i]
            }
        }
    
        for _, s := range scans {
            if s.WorkerID == nil || s.Equipment == nil || s.ScanTime == nil {
                continue
            }
            w := workerMap[*s.WorkerID]
            if w == nil {
                continue
            }
    
            t := true
            switch *s.Equipment {
            case "Hat":
                w.HatStatus = &t
            case "Shirt":
                w.ShirtStatus = &t
            case "Boot":
                w.BootStatus = &t
            case "Glove":
                w.GloveStatus = &t
            }
    
            if w.LastestScan == nil || s.ScanTime.After(*w.LastestScan) {
                w.LastestScan = s.ScanTime
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
        var workers []Worker

        loc, _ := time.LoadLocation("Asia/Bangkok")
        today := time.Now().In(loc).Format("2006-01-02")

        err := DB.Where("DATE(lastest_scan) = ?", today).Find(&workers).Error
        if err != nil {
            return c.Status(500).JSON(fiber.Map{"error": err.Error()})
        }

        return c.JSON(workers)
    })

    port := os.Getenv("PORT")
    if port == "" {
        port = "3000"
    }

    log.Println("Server starting on port:", port)
    log.Fatal(app.Listen(":" + port))
}