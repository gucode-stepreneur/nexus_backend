package main

import (
    "time"
    "log"
    "os"
    "github.com/gofiber/fiber/v2"
    "gorm.io/driver/mysql"
    "gorm.io/gorm"
    "github.com/gofiber/fiber/v2/middleware/cors"
)

type Worker struct {
    ID          *int       `json:"id"`
    Name        *string    `json:"name"`
    Position    *string    `json:"position"`
    HatID       *string    `json:"hat_id"`
    ShirtID     *string    `json:"shirt_id"`
    BootID      *string    `json:"boot_id"`
    GloveID     *string    `json:"glove_id"`
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

    // Auto migrate
    DB.AutoMigrate(&Worker{})

    // Fiber app
    app := fiber.New()

    app.Use(cors.New(cors.Config{
        AllowOrigins: "*",
        AllowMethods: "GET,POST,PUT,DELETE",
        AllowHeaders: "*",
    }))
    

    app.Get("/get_today_worker", func(c *fiber.Ctx) error {
        var workers []Worker

        loc, _ := time.LoadLocation("Asia/Bangkok")
        today := time.Now().In(loc).Format("2006-01-02")

        err := DB.Where("DATE(lastest_scan) = ?", today).Find(&workers).Error
        if err != nil {
            return c.Status(500).JSON(fiber.Map{"error": err.Error()})
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

    // -----------------------------
    // ❗ FIX: ใช้ PORT จาก Render
    // -----------------------------
    port := os.Getenv("PORT")
    if port == "" {
        port = "3000" // ใช้ตอนรันในเครื่อง
    }

    log.Println("Listening on port:", port)
    log.Fatal(app.Listen(":" + port))
}
