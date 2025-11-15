package main

import (
    "time"
    "log"
    "os"
    "github.com/gofiber/fiber/v2"
    "gorm.io/driver/mysql"
    "gorm.io/gorm"
    "github.com/gofiber/fiber/v2/middleware/cors"
    "github.com/gofiber/fiber/v2/middleware/limiter"
)

type Worker struct {
    ID          *int       `json:"id"`
    Name        *string    `json:"name"`
    Position    *string    `json:"position"`

    HatID       *string    `json:"hat_id"`
    HatStatus   bool       `json:"hat_status" gorm:"default:false"`

    ShirtID     *string    `json:"shirt_id"`
    ShirtStatus bool       `json:"shirt_status" gorm:"default:false"`

    BootID      *string    `json:"boot_id"`
    BootStatus  bool       `json:"boot_status" gorm:"default:false"`

    GloveID     *string    `json:"glove_id"`
    GloveStatus bool       `json:"glove_status" gorm:"default:false"`

    CreatedAt   *time.Time `json:"created_at"`
    LastestScan *time.Time `json:"lastest_scan"`
}

type Scan struct {
    ID           *int       `json:"id"`
    WorkerID     *int       `json:"worker_id"`
    ScanDate     *time.Time `json:"scan_date"`
    ScanTime     *time.Time `json:"scan_time"`
    ScannedNFCID *string    `json:"scanned_nfc_id"`
    Status       *string    `json:"status"`
    Equipment    *string    `json:"equip_name"`
}

var DB *gorm.DB

func main() {
    // Connect DB
    dsn := "root:zTuGFSJnzSDtQQexCsJnakBWFIHUhCbH@tcp(shortline.proxy.rlwy.net:11710)/railway?charset=utf8mb4&parseTime=True&loc=Local"
    db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
    if err != nil {
        log.Fatal("DB connection failed:", err)
    }
    DB = db

    // Auto migrate + สร้าง Index (ทำครั้งเดียว)
    DB.AutoMigrate(&Worker{}, &Scan{})
    createIndexes()

    // Fiber app
    app := fiber.New()

    app.Use(cors.New(cors.Config{
        AllowOrigins: "*",
        AllowMethods: "GET,POST,PUT,DELETE",
        AllowHeaders: "*",
    }))
    app.Use(limiter.New(limiter.Config{
        Max:        1,
        Expiration: 4 * time.Second,
    }))

    // =============================
    // OPTIMIZED: /get_today_worker
    // =============================
    app.Get("/get_today_worker", func(c *fiber.Ctx) error {
        loc, _ := time.LoadLocation("Asia/Bangkok")
        now := time.Now().In(loc)
        todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
        todayEnd := todayStart.Add(24 * time.Hour)

        var workers []Worker

        // ใช้ Raw SQL + JOIN + Conditional Aggregation
        err := DB.Raw(`
            SELECT 
                w.id,
                w.name,
                w.position,
                w.hat_id,
                w.shirt_id,
                w.boot_id,
                w.glove_id,
                w.created_at,
                
                MAX(s.scan_time) as lastest_scan,
                
                COALESCE(MAX(CASE WHEN s.equipment = 'Hat'   THEN 1 ELSE 0 END), 0) = 1 as hat_status,
                COALESCE(MAX(CASE WHEN s.equipment = 'Shirt' THEN 1 ELSE 0 END), 0) = 1 as shirt_status,
                COALESCE(MAX(CASE WHEN s.equipment = 'Boot'  THEN 1 ELSE 0 END), 0) = 1 as boot_status,
                COALESCE(MAX(CASE WHEN s.equipment = 'Glove' THEN 1 ELSE 0 END), 0) = 1 as glove_status

            FROM workers w
            LEFT JOIN scans s 
                ON w.id = s.worker_id 
                AND s.scan_date >= ? 
                AND s.scan_date < ?
            GROUP BY w.id
        `, todayStart, todayEnd).Scan(&workers).Error

        if err != nil {
            return c.Status(500).JSON(fiber.Map{"error": err.Error()})
        }

        return c.JSON(workers)
    })

    // =============================
    // /get_all_worker (เร็วอยู่แล้ว)
    // =============================
    app.Get("/get_all_worker", func(c *fiber.Ctx) error {
        var workers []Worker
        if err := DB.Find(&workers).Error; err != nil {
            return c.Status(500).JSON(fiber.Map{"error": err.Error()})
        }
        return c.JSON(workers)
    })

    // =============================
    // OPTIMIZED: /get_scan
    // =============================
    app.Get("/get_scan", func(c *fiber.Ctx) error {
        loc, _ := time.LoadLocation("Asia/Bangkok")
        now := time.Now().In(loc)
        todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
        todayEnd := todayStart.Add(24 * time.Hour)

        var workers []Worker
        err := DB.Where("lastest_scan >= ? AND lastest_scan < ?", todayStart, todayEnd).
            Find(&workers).Error

        if err != nil {
            return c.Status(500).JSON(fiber.Map{"error": err.Error()})
        }

        return c.JSON(workers)
    })

    // =============================
    // PORT
    // =============================
    port := os.Getenv("PORT")
    if port == "" {
        port = "3000"
    }
    log.Println("Listening on port:", port)
    log.Fatal(app.Listen(":" + port))
}

// =============================
// สร้าง Index (ครั้งเดียว)
// =============================
func createIndexes() {
    // ทำแค่ครั้งเดียว ถ้ามีแล้วจะไม่ error
    DB.Exec("CREATE INDEX IF NOT EXISTS idx_scans_scan_date ON scans(scan_date)")
    DB.Exec("CREATE INDEX IF NOT EXISTS idx_scans_worker_id ON scans(worker_id)")
    DB.Exec("CREATE INDEX IF NOT EXISTS idx_workers_lastest_scan ON workers(lastest_scan)")
}