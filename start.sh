#!/bin/bash

echo "Mematikan service SERVO sebelumnya (jika ada)..."
pkill -f "servo" || true
sleep 1

echo "Mem-build ulang file binary SERVO..."
go build -o servo .

echo "Menjalankan SERVO di mode latar belakang (background)..."
nohup ./servo --port :8080 > servo.log 2>&1 &

echo "Selesai! SERVO sedang berjalan di port 8080."
echo "Untuk melihat log aktivitas, jalankan perintah: tail -f servo.log"
