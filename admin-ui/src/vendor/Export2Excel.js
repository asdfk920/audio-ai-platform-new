import XLSX from 'xlsx'
import FileSaver from 'file-saver'

/**
 * Excel导出工具（供后台各列表页面“导出”按钮使用）
 *
 * 页面侧会动态 import('@/vendor/Export2Excel')，然后调用：
 * excel.export_json_to_excel({ header, data, filename, autoWidth, bookType })
 *
 * 其中 `data` 已经是二维数组（由 formatJson 生成），shape: Array<Array<any>>
 */
export function export_json_to_excel({ header, data, filename = '导出数据', autoWidth, bookType = 'xlsx' } = {}) {
  const safeHeader = Array.isArray(header) ? header : []
  const safeData = Array.isArray(data) ? data : []

  // 将表头+数据组合为 AOA（二维数组），再转 sheet
  const aoa = [safeHeader, ...safeData]
  const ws = XLSX.utils.aoa_to_sheet(aoa)

  const wb = XLSX.utils.book_new()
  XLSX.utils.book_append_sheet(wb, ws, 'Sheet1')

  const out = XLSX.write(wb, {
    bookType: bookType || 'xlsx',
    type: 'array'
  })

  const blob = new Blob([out], { type: 'application/octet-stream' })
  FileSaver.saveAs(blob, `${filename}.${bookType || 'xlsx'}`)
}

export default {
  export_json_to_excel
}

