import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { formatNumber } from "@/lib/format"

export type ResultFilterValue = "all" | "approved" | "rejected"

interface Props {
  value: ResultFilterValue
  onChange: (value: ResultFilterValue) => void
  counts?: { all: number; approved: number; rejected: number }
}

/** Filtro por resultado: todas, só as aprovadas ou só as reprovadas (com a contagem de cada uma, se vier). */
export function ResultFilter({ value, onChange, counts }: Props) {
  const items: { value: ResultFilterValue; label: string }[] = [
    { value: "all", label: "Todas" },
    { value: "approved", label: "Aprovadas" },
    { value: "rejected", label: "Reprovadas" },
  ]

  return (
    <Tabs value={value} onValueChange={(next) => onChange(next as ResultFilterValue)}>
      <TabsList aria-label="Filtrar por resultado" className="w-full sm:w-fit">
        {items.map((item) => (
          <TabsTrigger key={item.value} value={item.value} className="flex-1 sm:flex-none">
            {item.label}
            {counts && <span className="ml-1 text-xs opacity-70">{formatNumber(counts[item.value])}</span>}
          </TabsTrigger>
        ))}
      </TabsList>
    </Tabs>
  )
}
