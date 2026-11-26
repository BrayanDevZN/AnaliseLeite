import { CircleCheckIcon, CircleXIcon } from "lucide-react"

import { Badge } from "@/components/ui/badge"

/** "aprovado" ou "reprovado", a resposta do modelo e do exame. */
export function StatusBadge({ value }: { value: string }) {
  const approved = value.toLowerCase() === "aprovado"

  return (
    <Badge variant={approved ? "success" : "destructive"}>
      {approved ? <CircleCheckIcon data-icon="inline-start" /> : <CircleXIcon data-icon="inline-start" />}
      {approved ? "Aprovado" : "Reprovado"}
    </Badge>
  )
}
