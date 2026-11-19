-- =====================================================
-- DATABASE: pemweb
-- =====================================================
CREATE DATABASE IF NOT EXISTS pemweb
    CHARACTER SET utf8mb4
    COLLATE utf8mb4_unicode_ci;

USE pemweb;

-- =====================================================
-- TABEL: anggota
-- =====================================================
CREATE TABLE IF NOT EXISTS anggota (
    id_anggota     INT AUTO_INCREMENT PRIMARY KEY,
    no_anggota     VARCHAR(20)  NOT NULL UNIQUE,
    nama           VARCHAR(100) NOT NULL,
    alamat         VARCHAR(255),
    no_hp          VARCHAR(20),
    tanggal_daftar DATE         NOT NULL,
    status         ENUM('Aktif','Keluar') NOT NULL DEFAULT 'Aktif'
) ENGINE=InnoDB;

-- =====================================================
-- TABEL: simpanan
-- =====================================================
CREATE TABLE IF NOT EXISTS simpanan (
    id_simpanan INT AUTO_INCREMENT PRIMARY KEY,
    id_anggota  INT NOT NULL,
    jenis       ENUM('Pokok','Wajib','Sukarela') NOT NULL,
    tanggal     DATE NOT NULL,
    nominal     BIGINT UNSIGNED NOT NULL,
    CONSTRAINT fk_simpanan_anggota
        FOREIGN KEY (id_anggota) REFERENCES anggota(id_anggota)
        ON UPDATE CASCADE ON DELETE RESTRICT
) ENGINE=InnoDB;

-- =====================================================
-- TABEL: penarikan_sukarela
-- =====================================================
CREATE TABLE IF NOT EXISTS penarikan_sukarela (
    id_penarikan INT AUTO_INCREMENT PRIMARY KEY,
    id_anggota   INT NOT NULL,
    tanggal      DATE NOT NULL,
    nominal      BIGINT UNSIGNED NOT NULL,
    CONSTRAINT fk_penarikan_anggota
        FOREIGN KEY (id_anggota) REFERENCES anggota(id_anggota)
        ON UPDATE CASCADE ON DELETE RESTRICT
) ENGINE=InnoDB;

-- =====================================================
-- TABEL: pinjaman
-- =====================================================
CREATE TABLE IF NOT EXISTS pinjaman (
    id_pinjaman       INT AUTO_INCREMENT PRIMARY KEY,
    id_anggota        INT NOT NULL,
    tanggal_pengajuan DATE NOT NULL,
    tanggal_cair      DATE,
    jumlah_pinjaman   BIGINT UNSIGNED NOT NULL,
    tenor             INT NOT NULL COMMENT 'dalam bulan',
    jasa_per_bulan    BIGINT UNSIGNED NOT NULL,
    status            ENUM('Aktif','Lunas') NOT NULL DEFAULT 'Aktif',
    CONSTRAINT fk_pinjaman_anggota
        FOREIGN KEY (id_anggota) REFERENCES anggota(id_anggota)
        ON UPDATE CASCADE ON DELETE RESTRICT
) ENGINE=InnoDB;

-- =====================================================
-- TABEL: jadwal_angsuran
-- =====================================================
CREATE TABLE IF NOT EXISTS jadwal_angsuran (
    id_jadwal    INT AUTO_INCREMENT PRIMARY KEY,
    id_pinjaman  INT NOT NULL,
    angsuran_ke  INT NOT NULL,
    jatuh_tempo  DATE NOT NULL,
    pokok        BIGINT UNSIGNED NOT NULL,
    jasa         BIGINT UNSIGNED NOT NULL,
    status       ENUM('Lunas','Belum') NOT NULL DEFAULT 'Belum',
    UNIQUE KEY uq_pinjaman_angsuran (id_pinjaman, angsuran_ke),
    CONSTRAINT fk_jadwal_pinjaman
        FOREIGN KEY (id_pinjaman) REFERENCES pinjaman(id_pinjaman)
        ON UPDATE CASCADE ON DELETE CASCADE
) ENGINE=InnoDB;

-- =====================================================
-- TABEL: pembayaran
-- =====================================================
CREATE TABLE IF NOT EXISTS pembayaran (
    id_pembayaran INT AUTO_INCREMENT PRIMARY KEY,
    id_jadwal     INT NOT NULL,
    tanggal_bayar DATE NOT NULL,
    total_bayar   BIGINT UNSIGNED NOT NULL,
    denda_dibayar BIGINT UNSIGNED NOT NULL DEFAULT 0,
    jasa_dibayar  BIGINT UNSIGNED NOT NULL DEFAULT 0,
    pokok_dibayar BIGINT UNSIGNED NOT NULL DEFAULT 0,
    CONSTRAINT fk_pembayaran_jadwal
        FOREIGN KEY (id_jadwal) REFERENCES jadwal_angsuran(id_jadwal)
        ON UPDATE CASCADE ON DELETE RESTRICT
) ENGINE=InnoDB;

-- =====================================================
-- TABEL: pelunasan
-- =====================================================
CREATE TABLE IF NOT EXISTS pelunasan (
    id_pelunasan INT AUTO_INCREMENT PRIMARY KEY,
    id_pinjaman  INT NOT NULL UNIQUE,
    tanggal      DATE NOT NULL,
    sisa_pokok   BIGINT UNSIGNED NOT NULL,
    jasa         BIGINT UNSIGNED NOT NULL DEFAULT 0,
    administrasi BIGINT UNSIGNED NOT NULL DEFAULT 0,
    total        BIGINT UNSIGNED NOT NULL,
    CONSTRAINT fk_pelunasan_pinjaman
        FOREIGN KEY (id_pinjaman) REFERENCES pinjaman(id_pinjaman)
        ON UPDATE CASCADE ON DELETE RESTRICT
) ENGINE=InnoDB;

-- =====================================================
-- TABEL: admin
-- =====================================================
CREATE TABLE IF NOT EXISTS admin (
    id_admin      INT AUTO_INCREMENT PRIMARY KEY,
    username      VARCHAR(50)  NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    nama          VARCHAR(100) NOT NULL
) ENGINE=InnoDB;
