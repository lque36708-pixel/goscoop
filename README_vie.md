# goscoop

Trình CLI viết bằng Go thay thế backend PowerShell của Scoop. Tương thích với bucket và manifest có sẵn của Scoop. Một file nhị phân duy nhất, không phụ thuộc runtime, cài đặt nhanh hơn.

## Cài đặt

### Một dòng lệnh (cmd)

**Cài độc lập** (tạo `~\goscoop\` và thêm vào PATH):
```cmd
md "%USERPROFILE%\goscoop" && curl -Lo "%USERPROFILE%\goscoop\goscoop.exe" https://github.com/lque36708-pixel/goscoop/releases/latest/download/goscoop.exe && setx PATH "%PATH%;%USERPROFILE%\goscoop"
```

**Nếu bạn đã có Scoop** (đặt vào thư mục shims của Scoop, đã có trong PATH):
```cmd
curl -Lo "%USERPROFILE%\scoop\shims\goscoop.exe" https://github.com/lque36708-pixel/goscoop/releases/latest/download/goscoop.exe
```

Không cần quyền admin. Khởi động lại terminal sau khi chạy `setx`.

### Qua `go install`

```bash
go install github.com/lque36708-pixel/goscoop@latest
```

### Từ mã nguồn

```bash
git clone https://github.com/lque36708-pixel/goscoop.git
cd goscoop
go build -o goscoop.exe .
```

## Sử dụng

```
goscoop search chrome
goscoop install googlechrome
goscoop list
goscoop update
goscoop uninstall googlechrome
```

## Lệnh

| Lệnh | Hành động |
|---|---|
| `install <app>` | Cài đặt ứng dụng (tải đa luồng, tự động giải nén, persist, shim, nén LZX) |
| `update [app]` | Cập nhật tất cả bucket / một ứng dụng cụ thể |
| `uninstall <app> [apps...]` | Gỡ ứng dụng (`-p` để xoá luôn dữ liệu persist; `--self` để gỡ hoàn toàn goscoop) |
| `list` | Xem danh sách ứng dụng đã cài |
| `search <query>` | Tìm kiếm ứng dụng trong tất cả bucket (tự động cache sau lần đầu) |
| `status` | Kiểm tra ứng dụng cần cập nhật (tôn trọng `.hold`) |
| `info <app>` | Xem chi tiết manifest |
| `bucket list\|add\|rm` | Quản lý bucket |
| `cache list\|rm [app]` | Quản lý bộ nhớ đệm tải về |
| `hold/unhold <app>` | Giữ ứng dụng ở phiên bản hiện tại (ngăn cập nhật) |
| `reset <app>` | Cài đặt lại shim |
| `optimize` | Nén tất cả ứng dụng bằng LZX |
| `upgrade` | Tự cập nhật goscoop lên bản mới nhất |
| `--global`/`-g` | Cài đặt vào `%ProgramData%\scoop` |

## So sánh tính năng

| Tính năng | Scoop (PS) | goscoop |
|---|---|---|
| Một file nhị phân | | ✓ |
| Không phụ thuộc runtime | | ✓ |
| Tải đa luồng | | ✓ (4 phần mỗi file) |
| Thanh tiến trình hoạt ảnh | | ✓ |
| Tự động nén LZX khi cài | | ✓ |
| Lệnh `optimize` | | ✓ |
| Shortcut trong Start Menu | ✓ | ✓ |
| `depends` trong manifest | ✓ | ✓ |
| Persist (thư mục + file) | ✓ | ✓ |
| Giải nén lồng nhau | ✓ | ✓ |
| Innosetup / MSI / 7z / tar | ✓ | ✓ |
| Script pre/post install | ✓ | ✓ |
| Quản lý bucket | ✓ | ✓ |
| Hold / unhold | ✓ | ✓ |
| Search cache / index | | ✓ |
| `list` / `search` / `status` | ✓ | ✓ |
| `cache` management | ✓ | ✓ |
| Hỗ trợ `--global` | ✓ | ✓ |
| Gợi ý tên tương tự khi gõ sai | | ✓ |
| Cảnh báo ứng dụng vẫn còn sau gỡ | | ✓ |
