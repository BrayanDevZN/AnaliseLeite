import { useState } from "react"
import { DownloadIcon } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Spinner } from "@/components/ui/spinner"
import { notify } from "@/lib/notify"
import { downloadReport, type ReportData } from "@/lib/report"

interface Props {
  /** Chamado na hora do clique, com os dados já atuais (assim a tela não precisa montar o relatório antes). */
  data: () => ReportData
  disabled?: boolean
  size?: "default" | "lg"
  className?: string
}

/** Baixa o relatório da análise em PDF. A biblioteca do PDF só é carregada quando se clica. */
export function ReportButton({ data, disabled, size = "default", className }: Props) {
  const [downloading, setDownloading] = useState(false)

  async function download() {
    setDownloading(true)

    try {
      await downloadReport(data())
    } catch {
      notify.error("Não foi possível gerar o PDF", "Tente de novo em instantes.")
    } finally {
      setDownloading(false)
    }
  }

  return (
    <Button variant="outline" size={size} className={className} onClick={download} disabled={disabled || downloading} aria-label="Baixar relatório em PDF">
      {downloading ? <Spinner data-icon="inline-start" /> : <DownloadIcon data-icon="inline-start" />}
      Baixar PDF
    </Button>
  )
}
