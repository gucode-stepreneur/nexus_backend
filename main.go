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
        HatStatus   *bool      `json:"hat_status" gorm:"default:false"`

        ShirtID     *string    `json:"shirt_id"`
        ShirtStatus *bool      `json:"shirt_status" gorm:"default:false"`

        BootID      *string    `json:"boot_id"`
        BootStatus  *bool      `json:"boot_status" gorm:"default:false"`

        GloveID     *string    `json:"glove_id"`
        GloveStatus *bool      `json:"glove_status" gorm:"default:false"`

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
        DB.AutoMigrate(&Worker{}, &Scan{})
        

        // Fiber app
        app := fiber.New()

        app.Use(cors.New(cors.Config{
            AllowOrigins: "*",
            AllowMethods: "GET,POST,PUT,DELETE",
            AllowHeaders: "*",
        }))
        app.Use(limiter.New(limiter.Config{
            Max:        1,                    // ยิงได้ 1 ครั้ง
            Expiration: 4 * time.Second,      // ทุก 4 วิ
        }))
        

        app.Get("/get_today_worker", func(c *fiber.Ctx) error {
            var workers []Worker
        
            loc, _ := time.LoadLocation("Asia/Bangkok")
            now := time.Now().In(loc)
            todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
            todayEnd := todayStart.Add(24 * time.Hour)
        
            // Workers
            if err := DB.Find(&workers).Error; err != nil {
                return c.Status(500).JSON(fiber.Map{"error": err.Error()})
            }
        
            // Scans (FAST)
            var scans []Scan
            err := DB.Where("scan_date >= ? AND scan_date < ?", todayStart, todayEnd).Find(&scans).Error
            if err != nil {
                return c.Status(500).JSON(fiber.Map{"error": err.Error()})
            }
        
            workerMap := map[int]*Worker{}
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
        
                if w.LastestScan == nil || s.ScanTime.Before(*w.LastestScan) {
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
