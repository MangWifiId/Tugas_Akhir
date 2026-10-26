
document.addEventListener("DOMContentLoaded", () => {
    const pageTitle = document.getElementById("page-title");
    const dashboard = document.getElementById("dashboard");
    const modulePage = document.getElementById("module-page");
    const moduleHeading = document.getElementById("module-heading");
    const moduleDescription = document.getElementById("module-description");
    const moduleSymbol = document.getElementById("module-symbol");
    const dateLabel = document.getElementById("current-date");
    const navLinks = document.querySelectorAll(".nav-link");

    const modules = {
        inventory: {
            title: "Inventaris",
            description:
                "Kelola spare part, jumlah stok, harga jual, dan HPP.",
            symbol: "▦"
        },
        purchases: {
            title: "Pembelian",
            description:
                "Catat pembelian spare part dan penerimaan barang.",
            symbol: "↓"
        },
        sales: {
            title: "Penjualan",
            description:
                "Kelola transaksi penjualan spare part secara tunai.",
            symbol: "↗"
        },
        services: {
            title: "Servis Bengkel",
            description:
                "Kelola pekerjaan servis, mekanik, dan pemakaian spare part.",
            symbol: "🔧"
        },
        customers: {
            title: "Pelanggan",
            description:
                "Kelola informasi pelanggan dan kendaraan mereka.",
            symbol: "♙"
        }
    };

    // Menampilkan tanggal sesuai lokal perangkat.
    if (dateLabel) {
        dateLabel.textContent = new Intl.DateTimeFormat("id-ID", {
            weekday: "long",
            day: "numeric",
            month: "long",
            year: "numeric"
        }).format(new Date());
    }

    function showDashboard() {
        dashboard.classList.remove("hidden");
        modulePage.classList.add("hidden");
        pageTitle.textContent = "Dashboard";

        navLinks.forEach((link) => {
            link.classList.toggle(
                "active",
                link.getAttribute("href") === "#dashboard"
            );
        });
    }

    function showModule(moduleKey) {
        const module = modules[moduleKey];

        if (!module) {
            showDashboard();
            return;
        }

        dashboard.classList.add("hidden");
        modulePage.classList.remove("hidden");

        pageTitle.textContent = module.title;
        moduleHeading.textContent = module.title;
        moduleDescription.textContent = module.description;
        moduleSymbol.textContent = module.symbol;

        navLinks.forEach((link) => {
            link.classList.toggle(
                "active",
                link.getAttribute("href") === `#${moduleKey}`
            );
        });
    }

    function navigate() {
        const page = window.location.hash.slice(1);

        if (!page || page === "dashboard") {
            showDashboard();
            return;
        }

        showModule(page);
    }

    // Navigasi sidebar dan kartu akses cepat.
    window.addEventListener("hashchange", navigate);

    // Buka dashboard melalui tombol kembali.
    const backLink = document.querySelector(".back-link");

    if (backLink) {
        backLink.addEventListener("click", () => {
            // Navigasi normal ditangani oleh hashchange.
        });
    }

    navigate();
});
