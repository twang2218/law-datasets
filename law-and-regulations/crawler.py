# ❯ curl 'https://flk.npc.gov.cn/api/?type=&searchType=title%3Bvague&sortTr=f_bbrq_s%3Bdesc&gbrqStart=&gbrqEnd=&sxrqStart=&sxrqEnd=&page=1&size=10&_=1693416204292' \
#     --compressed \
#     -H 'User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:109.0) Gecko/20100101 Firefox/115.0' \
#     -H 'Accept: application/json, text/javascript, */*; q=0.01' \
#     -H 'Accept-Language: en-US,en;q=0.5' \
#     -H 'Accept-Encoding: gzip, deflate, br' \
#     -H 'X-Requested-With: XMLHttpRequest' \
#     -H 'Connection: keep-alive' \
#     -H 'Referer: https://flk.npc.gov.cn/index.html' \
#     -H 'Cookie: cna=tPoqHU27DEgCAXZmWwQbkasU; yfx_c_g_u_id_10006696=_ck23072521120113841492621431773; yfx_f_l_v_t_10006696=f_t_1690283521384__r_t_1690283521384__v_t_1690283521384__r_c_0; Hm_lvt_54434aa6770b6d9fef104d146430b53b=1692540211; Hm_lpvt_54434aa6770b6d9fef104d146430b53b=1693416206; acw_tc=2ff6299516934158704976331ebf19857db303cd243ecee11e5718a9c2' \
#     -H 'Sec-Fetch-Dest: empty' \
#     -H 'Sec-Fetch-Mode: cors' \
#     -H 'Sec-Fetch-Site: same-origin' \
#     -H 'Pragma: no-cache' \
#     -H 'Cache-Control: no-cache' \
#     -H 'TE: trailers'


import requests
import json
import time
import random
import os
from pprint import pprint
from loguru import logger
import tempfile
import pypandoc
from tqdm import tqdm

def get_data_from_flknpc(page=1, size:int=10):
    url = "https://flk.npc.gov.cn/api/"
    headers = {
        'User-Agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:109.0) Gecko/20100101 Firefox/115.0',
        'Accept': 'application/json, text/javascript, */*; q=0.01',
        'Accept-Language': 'en-US,en;q=0.5',
        'X-Requested-With': 'XMLHttpRequest',
        'Referer': 'https://flk.npc.gov.cn/index.html',
        'Pragma': 'no-cache',
        'Cache-Control': 'no-cache',
    }
    params = {
        'type': '',
        'searchType': 'title;vague',
        'sortTr': 'f_bbrq_s;desc',
        'gbrqStart': '',
        'gbrqEnd': '',
        'sxrqStart': '',
        'sxrqEnd': '',
        'page': page,
        'size': size,
        '_': 1693416204292,
    }

    response = requests.get(url, headers=headers, params=params)
    
    if response.status_code == 200:
        return response.json()
    else:
        print("Error:", response.status_code)
        return None

def get_detail(id):
    url = "https://flk.npc.gov.cn/api/detail"
    data = {
        'id': id,
    }
    response = requests.post(url, data=data, timeout=10)
    if response.status_code == 200:
        return response.json()
    else:
        print("Error:", response.status_code)
        return None

def get_links(law):
    logger.info(f"正在获取 {law['title']} 的链接")
    detail = get_detail(law['id'])
    pprint(detail)
    if detail and detail['success']:
        docs = detail['result']['body']
        links = {}
        for doc in docs:
            path = doc['path'] if doc['path'] else doc['url']
            link = f"https://wb.flk.npc.gov.cn{path}"
            links[doc['type']] = link
        return links

def get_markdown(law):
    logger.info(f"正在获取 {law['title']} 的内容，并以 markdown 格式返回")
    links = law['links']
    if 'WORD' in links:
        format = 'docx'
        link = links['WORD']
        to = 'markdown'
    elif 'HTML' in links:
        format = 'html'
        link = links['HTML']
        to = 'plain'
    elif 'PDF' in links:
        format = 'pdf'
        link = links['PDF']
        to = 'markdown'

    with tempfile.NamedTemporaryFile() as f:
        response = requests.get(link, timeout=10)
        f.write(response.content)
        f.flush()
        output = pypandoc.convert_file(f.name,
                                       to=to,
                                       format=format,
                                       extra_args=['--extract-media', '.'])
        return output

if __name__ == "__main__":
    first_page = get_data_from_flknpc()
    pprint(first_page)
    laws = []
    if first_page and first_page['success']:
        result = first_page['result']
        print(f'共{result["totalSizes"]}条数据，当前第{result["page"]}页，每页{result["size"]}条')
        total_pages = result['totalSizes'] // result['size'] + 1
        data = result['data']
        for item in data:
            item['links'] = get_links(item)
            item['content'] = get_markdown(item)
        laws.extend(result['data'])
    total_pages = 3
    for page in tqdm(range(2, total_pages+1)):
        time.sleep(random.randint(1, 5))
        data = get_data_from_flknpc(page=page)
        if data and data['success']:
            result = data['result']
            data = result['data']
            for item in data:
                item['links'] = get_links(item)
                item['content'] = get_markdown(item)
            laws.extend(data)

            # 保存文件
            with open(os.path.join(os.path.dirname(__file__), 'laws.json'), 'w') as f:
                json.dump(laws, f, ensure_ascii=False, indent=2)

