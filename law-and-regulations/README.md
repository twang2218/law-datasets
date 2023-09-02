# 中国法律法规数据库

## 数据集介绍

本数据集包含了从中国法律法规数据库(https://flk.npc.gov.cn/)中获取的几乎所有法律法规数据。截止 2023年9月1日，共计 22556 条法律法规，本数据集包含了其中 22552 条法律法规。

## 数据集内容

为了避免空间浪费，下载的JSON文件进行了压缩。压缩和解压缩直接使用 `make` 命令即可。

```bash
# 压缩
make zip

# 解压缩
make unzip
```

## 实现方式

网站爬虫使用了两种实现方式，首先使用了 Python 进行逻辑验证，然后使用 Go 重新实现，并支持并发、断点续传等功能，让程序更加健壮。

要执行爬虫，可以执行：

```bash
go run crawler.go
```

或者使用 Makefile:

```bash
make fetch
```

### 爬虫实现

#### 法规列表

我们通过访问 `https://flk.npc.gov.cn/api/` API，请求参数中指定 `page`，`page` 从 `1` 开始递增，直到最大页数。

返回的内容为 JSON 格式， 如：

```json
{'code': 200,
 'message': 'success',
 'result': {'data': [{'expiry': '2018-03-11 00:00:00',
                      'id': 'MmM5MDlmZGQ2NzhiZjE3OTAxNjc4YmY1YTQ4MzAwNGI%3D',
                      'office': '全国人民代表大会',
                      'publish': '2018-03-11 00:00:00',
                      'status': '1',
                      'title': '中华人民共和国宪法（2018年修正文本）',
                      'type': '宪法',
                      'url': 'https://flk.npc.gov.cn/detail2.html?MmM5MDlmZGQ2NzhiZjE3OTAxNjc4YmY1YTQ4MzAwNGI%3D'},
                     {'expiry': '2018-03-11 00:00:00',
                      'id': 'MmM5MDlmZGQ2NzhiZjE3OTAxNjc4YmY1OWRhODAwMmQ%3D',
                      'office': '全国人民代表大会',
                      'publish': '2018-03-11 00:00:00',
                      'status': '1',
                      'title': '中华人民共和国宪法修正案（2018年）',
                      'type': '宪法',
                      'url': 'https://flk.npc.gov.cn/detail2.html?MmM5MDlmZGQ2NzhiZjE3OTAxNjc4YmY1OWRhODAwMmQ%3D'},
                     {'expiry': '2004-03-14 00:00:00',
                      'id': 'MmM5MDlmZGQ2NzhiZjE3OTAxNjc4YmY1OWQ0MjAwMjM%3D',
                      'office': '全国人民代表大会',
                      'publish': '2004-03-14 00:00:00',
                      'status': '1',
                      'title': '中华人民共和国宪法修正案（2004年）',
                      'type': '宪法',
                      'url': 'https://flk.npc.gov.cn/detail2.html?MmM5MDlmZGQ2NzhiZjE3OTAxNjc4YmY1OWQ0MjAwMjM%3D'},
                     {'expiry': '1999-03-15 00:00:00',
                      'id': 'MmM5MDlmZGQ2NzhiZjE3OTAxNjc4YmY1OWM5ZTAwMTk%3D',
                      'office': '全国人民代表大会',
                      'publish': '1999-03-15 00:00:00',
                      'status': '1',
                      'title': '中华人民共和国宪法修正案（1999年）',
                      'type': '宪法',
                      'url': 'https://flk.npc.gov.cn/detail2.html?MmM5MDlmZGQ2NzhiZjE3OTAxNjc4YmY1OWM5ZTAwMTk%3D'},
                     {'expiry': '1993-03-29 00:00:00',
                      'id': 'MmM5MDlmZGQ2NzhiZjE3OTAxNjc4YmY1OWMwZDAwMGY%3D',
                      'office': '全国人民代表大会',
                      'publish': '1993-03-29 00:00:00',
                      'status': '1',
                      'title': '中华人民共和国宪法修正案（1993年）',
                      'type': '宪法',
                      'url': 'https://flk.npc.gov.cn/detail2.html?MmM5MDlmZGQ2NzhiZjE3OTAxNjc4YmY1OWMwZDAwMGY%3D'},
                     {'expiry': '1988-04-12 00:00:00',
                      'id': 'MmM5MDlmZGQ2NzhiZjE3OTAxNjc4YmY1OTQyNDAwMDU%3D',
                      'office': '全国人民代表大会',
                      'publish': '1988-04-12 00:00:00',
                      'status': '1',
                      'title': '中华人民共和国宪法修正案（1988年）',
                      'type': '宪法',
                      'url': 'https://flk.npc.gov.cn/detail2.html?MmM5MDlmZGQ2NzhiZjE3OTAxNjc4YmY1OTQyNDAwMDU%3D'},
                     {'expiry': '1982-12-04 00:00:00',
                      'id': 'MmM5MDlmZGQ2NzhiZjE3OTAxNjc4YmY1OWViODAwMzc%3D',
                      'office': '全国人民代表大会',
                      'publish': '1982-12-04 00:00:00',
                      'status': '1',
                      'title': '中华人民共和国宪法（1982年）',
                      'type': '宪法',
                      'url': 'https://flk.npc.gov.cn/detail2.html?MmM5MDlmZGQ2NzhiZjE3OTAxNjc4YmY1OWViODAwMzc%3D'},
                     {'expiry': '2023-07-01 00:00:00',
                      'id': 'ZmY4MDgxODE4OGQ3NDMwYjAxODkwMTljMTVlZTA5NDM%3D',
                      'office': '全国人民代表大会常务委员会',
                      'publish': '2023-06-28 00:00:00',
                      'status': '1',
                      'title': '中华人民共和国对外关系法',
                      'type': '法律',
                      'url': 'https://flk.npc.gov.cn/detail2.html?ZmY4MDgxODE4OGQ3NDMwYjAxODkwMTljMTVlZTA5NDM%3D'},
                     {'expiry': '2023-03-15 00:00:00',
                      'id': 'ZmY4MDgxODE4NjVlZGMxNDAxODZkOWQ2ZjFjYjI1MDM%3D',
                      'office': '全国人民代表大会',
                      'publish': '2023-03-13 00:00:00',
                      'status': '1',
                      'title': '中华人民共和国立法法',
                      'type': '法律',
                      'url': 'https://flk.npc.gov.cn/detail2.html?ZmY4MDgxODE4NjVlZGMxNDAxODZkOWQ2ZjFjYjI1MDM%3D'},
                     {'expiry': '2022-06-25 00:00:00',
                      'id': 'ZmY4MDgxODE4MjkxOTc3NzAxODI5ZjNiMjRmMzAwZDc%3D',
                      'office': '全国人民代表大会常务委员会',
                      'publish': '2022-06-24 00:00:00',
                      'status': '1',
                      'title': '中华人民共和国全国人民代表大会常务委员会议事规则',
                      'type': '法律',
                      'url': 'https://flk.npc.gov.cn/detail2.html?ZmY4MDgxODE4MjkxOTc3NzAxODI5ZjNiMjRmMzAwZDc%3D'}],
            'page': 1,
            'size': 10,
            'totalSizes': 22545},
 'success': True,
 'timestamp': 1693430843339}
```

#### 法规链接

上述列表中没有法规的链接信息，需要通过访问 `https://flk.npc.gov.cn/api/detail` API 来获得，其中的 `id` 参数为列表API返回条目中的 `id` 字段。

返回结果为 JSON，如：

```json
{'code': 200,
 'message': 'success',
 'result': {'bbbs': None,
            'body': [{'addr': '2c909fdd678bf17901678bf59e870030',
                      'mobile': '/xffl/html/99663651a9547da11c4019d71ab5ce2aae864769-mobile.html',
                      'path': '/xffl/PDF/c1c2617518b348f4977af282dfe30895.pdf',
                      'qr': '/xffl/PNG/c1c2617518b348f4977af282dfe30895.png',
                      'type': 'PDF',
                      'url': '/xffl/html/99663651a9547da11c4019d71ab5ce2aae864769.html'},
                     {'addr': '2c909fdd678bf17901678bf59eab0034',
                      'mobile': '/xffl/html/c35fb8c4f01b705befc1e257cc62d38397d9d193-mobile.html',
                      'path': '/xffl/WORD/0193ce7784e64a83b575aadb5e0b8622.docx',
                      'qr': '/xffl/PNG/0193ce7784e64a83b575aadb5e0b8622.png',
                      'type': 'WORD',
                      'url': '/xffl/html/c35fb8c4f01b705befc1e257cc62d38397d9d193.html'}],
            'dec': None,
            'expiry': '2018-03-11 00:00:00',
            'level': '宪法',
            'office': '全国人民代表大会',
            'otherFile': None,
            'publish': '2018-03-11 00:00:00',
            'status': '1',
            'title': '中华人民共和国宪法修正案（2018年）'},
 'success': True,
 'timestamp': 1693430847821}
```

需要注意的是，返回链接为相对路径，需要拼接上 `https://wb.flk.npc.gov.cn` 才能访问。

#### 法规内容

通过上述请求中所获得的链接信息，来下载法律内容。如果存在 WORD 文档，则优先使用 WORD 文档，否则使用 HTML 文档。

内容下载后，使用 pandoc 转换为 Markdown 格式。

```bash
# WORD
pandoc -f docx -t markdown

# HTML
pandoc -f html -t markdown
```

如果转换失败，则再尝试使用 tika 转换为纯文本。

```bash
tika -t <file>
```

#### 法规数据

获取的法规的元数据以及内容，会以 JSON 数组形式保存于 `laws.json` 文件。

如：

```json
[
  {
    "id": "ZmY4MDgwODE2ZjNjYmIzYzAxNmY0MGZkM2JiZjEwMjQ%3D",
    "title": "导游人员管理条例",
    "office": "国务院",
    "publish": "2017-10-07 00:00:00",
    "expiry": "",
    "type": "行政法规",
    "status": "有效",
    "url": "https://flk.npc.gov.cn/detail2.html?ZmY4MDgwODE2ZjNjYmIzYzAxNmY0MGZkM2JiZjEwMjQ%3D",
    "download_link_word": "https://wb.flk.npc.gov.cn/xzfg/WORD/be0c5465e9e24c8fb2c4b7e3f7e24d95.docx",
    "download_link_html": "",
    "download_link_pdf": "",
    "content": "导游人员管理条例\n\n(1999年5月14日中华人民共和国国务院令第263号发布　根据2017年10月7日《国务院关于修改部分行政法规的决定》修订)\n\n第一条　为了规范导游活动，保障旅游者和导游人员的合法权益，促进旅游业的健康发展，制定本条例。\n\n第二条　本条例所称导游人员，是指依照本条例的规定取得导游证，接受旅行社委派，为旅游者提供向导、讲解及相关旅游服务的人员。\n\n第三条　国家实行..."
  },
  ...
]
```


