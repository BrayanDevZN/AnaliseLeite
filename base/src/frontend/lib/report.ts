import type { Analysis, Content, Tank } from "@/lib/types"

type Row = Record<string, unknown>

export interface ReportData {
  analysis: Analysis
  farmName: string
  tank?: Tank
  /** O que foi enviado na análise (para uma amostra: os valores dela). */
  contents: Content[]
  /** Uma linha por amostra de um arquivo, com a resposta do modelo. Vazio se não deu para carregar. */
  rows?: Row[]
}

/** Quantas amostras entram na tabela do PDF: acima disso só vão as reprovadas (o PDF teria centenas de páginas). */
const MAX_ROWS = 400
const MAX_REJECTED = 1000

const number = new Intl.NumberFormat("pt-BR")
const dateTime = new Intl.DateTimeFormat("pt-BR", { dateStyle: "long", timeStyle: "short" })

const isApproved = (row: Row) => String(row.predict ?? "").toLowerCase() === "aprovado"
const cell = (value: unknown) => (value == null ? "-" : typeof value === "number" ? number.format(value) : String(value).replace(/T00:00:00.*$/, ""))

const COLUMNS: [string, string][] = [
  ["data", "Data"],
  ["temperatura", "Temp. (°C)"],
  ["ccs", "CCS"],
  ["cpp", "CPP"],
  ["tanque", "Tanque"],
  ["cidade", "Cidade"],
  ["estado", "UF"],
]

const SAMPLE_FIELDS: [string, string][] = [
  ["data", "Data da coleta"],
  ["temperature", "Temperatura (°C)"],
  ["ccs", "CCS"],
  ["cpp", "CPP"],
  ["tanque", "Código do tanque"],
  ["cidade", "Cidade"],
  ["state", "UF"],
]

// as cores do site (verde-pasto, vermelho, cinza) em RGB para o PDF
const GREEN: [number, number, number] = [18, 110, 76]
const RED: [number, number, number] = [200, 55, 50]
const GRAY: [number, number, number] = [95, 105, 100]

/** Monta o relatório em PDF e baixa. A biblioteca só é carregada aqui, para não pesar o resto do site. */
export async function downloadReport({ analysis, farmName, tank, contents, rows }: ReportData): Promise<void> {
  const [{ jsPDF }, { default: autoTable }] = await Promise.all([import("jspdf"), import("jspdf-autotable")])

  const doc = new jsPDF({ unit: "pt", format: "a4" })
  const width = doc.internal.pageSize.getWidth()
  const margin = 40
  let y = 0

  // faixa do topo
  doc.setFillColor(...GREEN)
  doc.rect(0, 0, width, 70, "F")
  doc.setTextColor(255, 255, 255)
  doc.setFont("helvetica", "bold")
  doc.setFontSize(20)
  doc.text("Análise Leite", margin, 34)
  doc.setFont("helvetica", "normal")
  doc.setFontSize(11)
  doc.text(`Relatório da análise #${analysis.id}`, margin, 54)
  y = 100

  // dados gerais
  doc.setTextColor(30, 40, 35)
  const info: [string, string][] = [
    ["Fazenda", farmName],
    ["Data", dateTime.format(new Date(analysis.created_at))],
    ["Tipo", analysis.kind === "file" ? "Arquivo com várias amostras" : analysis.kind === "image" ? "Imagem" : "Uma amostra"],
    ["Modelo", analysis.model || "Padrão"],
    ...(tank ? ([["Tanque", `${tank.type} (${number.format(tank.capacity_liters)} L)`]] as [string, string][]) : []),
  ]

  autoTable(doc, {
    startY: y,
    body: info,
    theme: "plain",
    styles: { fontSize: 10, cellPadding: 3 },
    columnStyles: { 0: { fontStyle: "bold", textColor: GRAY, cellWidth: 90 } },
    margin: { left: margin, right: margin },
  })

  y = (doc as unknown as { lastAutoTable: { finalY: number } }).lastAutoTable.finalY + 18

  const approved = rows ? rows.filter(isApproved).length : analysis.approved
  const rejected = rows ? rows.length - approved : analysis.rejected
  const total = approved + rejected

  doc.setFont("helvetica", "bold")
  doc.setFontSize(13)
  doc.text("Resultado", margin, y)
  y += 12

  if (total === 0) {
    doc.setFont("helvetica", "normal")
    doc.setFontSize(10)
    doc.setTextColor(...GRAY)
    doc.text("O resultado desta análise não foi guardado (ela é anterior a essa função).", margin, y + 14)
    y += 30
  } else if (analysis.kind === "sample" || total === 1) {
    const ok = approved > 0
    doc.setFillColor(...(ok ? GREEN : RED))
    doc.roundedRect(margin, y, width - margin * 2, 40, 6, 6, "F")
    doc.setTextColor(255, 255, 255)
    doc.setFontSize(15)
    doc.text(ok ? "Leite aprovado" : "Leite reprovado", margin + 14, y + 25)
    y += 56
  } else {
    const percent = Math.round((approved / total) * 100)
    const boxWidth = (width - margin * 2 - 20) / 3

    ;[
      { label: "Amostras", value: number.format(total), color: GRAY },
      { label: "Aprovadas", value: `${number.format(approved)} (${percent}%)`, color: GREEN },
      { label: "Reprovadas", value: `${number.format(rejected)} (${100 - percent}%)`, color: RED },
    ].forEach((box, index) => {
      const x = margin + index * (boxWidth + 10)

      doc.setDrawColor(...box.color)
      doc.setLineWidth(1)
      doc.roundedRect(x, y, boxWidth, 46, 6, 6, "S")
      doc.setTextColor(...GRAY)
      doc.setFont("helvetica", "normal")
      doc.setFontSize(9)
      doc.text(box.label, x + 10, y + 16)
      doc.setTextColor(...box.color)
      doc.setFont("helvetica", "bold")
      doc.setFontSize(13)
      doc.text(box.value, x + 10, y + 36)
    })

    // barra de aprovação
    const barY = y + 58

    doc.setFillColor(...RED)
    doc.roundedRect(margin, barY, width - margin * 2, 8, 4, 4, "F")
    doc.setFillColor(...GREEN)
    doc.roundedRect(margin, barY, Math.max(8, ((width - margin * 2) * approved) / total), 8, 4, 4, "F")
    y = barY + 28
  }

  // dados da amostra enviada
  const sample = contents.find((content) => content.kind === "content")?.content

  if (analysis.kind === "sample" && sample) {
    doc.setTextColor(30, 40, 35)
    doc.setFont("helvetica", "bold")
    doc.setFontSize(13)
    doc.text("Dados da amostra", margin, y)

    autoTable(doc, {
      startY: y + 8,
      body: SAMPLE_FIELDS.filter(([key]) => (sample as Record<string, unknown>)[key] != null).map(([key, label]) => [label, cell((sample as Record<string, unknown>)[key])]),
      theme: "striped",
      styles: { fontSize: 10, cellPadding: 4 },
      columnStyles: { 0: { fontStyle: "bold", cellWidth: 150 } },
      margin: { left: margin, right: margin },
    })
  }

  // tabela das amostras de um arquivo
  if ((analysis.kind === "file" || analysis.kind === "image") && rows && rows.length > 0) {
    const numbered = rows.map((row, index) => ({ row, number: index + 1 }))
    const all = numbered.length <= MAX_ROWS
    const shown = (all ? numbered : numbered.filter((item) => !isApproved(item.row))).slice(0, all ? MAX_ROWS : MAX_REJECTED)
    const available = COLUMNS.filter(([key]) => rows.some((row) => row[key] != null))

    doc.setTextColor(30, 40, 35)
    doc.setFont("helvetica", "bold")
    doc.setFontSize(13)
    doc.text(all ? "Amostras" : "Amostras reprovadas", margin, y)

    if (!all) {
      doc.setFont("helvetica", "normal")
      doc.setFontSize(9)
      doc.setTextColor(...GRAY)
      doc.text(
        `O arquivo tem ${number.format(rows.length)} amostras; para o PDF não ficar enorme, só as reprovadas estão listadas${rejected > MAX_REJECTED ? ` (as ${number.format(MAX_REJECTED)} primeiras)` : ""}.`,
        margin,
        y + 14,
        { maxWidth: width - margin * 2 },
      )
    }

    autoTable(doc, {
      startY: y + (all ? 8 : 24),
      head: [["#", ...available.map(([, label]) => label), "Resultado"]],
      body: shown.map((item) => [String(item.number), ...available.map(([key]) => cell(item.row[key])), isApproved(item.row) ? "Aprovado" : "Reprovado"]),
      theme: "striped",
      styles: { fontSize: 8, cellPadding: 3 },
      headStyles: { fillColor: GREEN },
      margin: { left: margin, right: margin },
      didParseCell: (data) => {
        if (data.section === "body" && data.column.index === available.length + 1) {
          data.cell.styles.textColor = data.cell.raw === "Aprovado" ? GREEN : RED
          data.cell.styles.fontStyle = "bold"
        }
      },
    })
  }

  // rodapé com a página
  const pages = doc.getNumberOfPages()
  const height = doc.internal.pageSize.getHeight()

  for (let page = 1; page <= pages; page++) {
    doc.setPage(page)
    doc.setFont("helvetica", "normal")
    doc.setFontSize(8)
    doc.setTextColor(...GRAY)
    doc.text(`Análise Leite - gerado em ${dateTime.format(new Date())}`, margin, height - 20)
    doc.text(`Página ${page} de ${pages}`, width - margin, height - 20, { align: "right" })
  }

  doc.save(`analise-${analysis.id}.pdf`)
}
