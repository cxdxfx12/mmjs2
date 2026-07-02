const BASE = ''

function token(): string {
  return localStorage.getItem('yunfei_token') || ''
}

export async function apiGet(path: string): Promise<any> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { Authorization: `Bearer ${token()}` },
  })
  return res.json()
}

export async function apiPost(path: string, body?: any): Promise<any> {
  const res = await fetch(`${BASE}${path}`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token()}`,
    },
    body: body ? JSON.stringify(body) : undefined,
  })
  return res.json()
}

export async function apiUpload(file: File, onProgress?: (pct: number) => void): Promise<any> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest()
    xhr.open('POST', `${BASE}/api/excel/upload`)
    xhr.setRequestHeader('Authorization', `Bearer ${token()}`)
    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable && onProgress) {
        onProgress(Math.round((e.loaded / e.total) * 100))
      }
    }
    xhr.onload = () => {
      try { resolve(JSON.parse(xhr.responseText)) }
      catch { reject(new Error('解析响应失败')) }
    }
    xhr.onerror = () => reject(new Error('上传失败'))
    const fd = new FormData()
    fd.append('file', file)
    xhr.send(fd)
  })
}

export async function apiExport(onProgress?: (pct: number) => void): Promise<Blob> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest()
    xhr.open('POST', `${BASE}/api/export`)
    xhr.setRequestHeader('Authorization', `Bearer ${token()}`)
    xhr.responseType = 'blob'
    xhr.onprogress = (e) => {
      if (e.lengthComputable && onProgress) {
        onProgress(Math.round((e.loaded / e.total) * 100))
      }
    }
    xhr.onload = () => {
      if (xhr.status === 200) {
        resolve(xhr.response)
      } else {
        reject(new Error('导出失败'))
      }
    }
    xhr.onerror = () => reject(new Error('导出失败'))
    xhr.send()
  })
}
