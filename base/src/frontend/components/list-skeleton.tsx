import { Skeleton } from "@/components/ui/skeleton"

/** Cartões em carregamento, no lugar de uma lista que ainda não chegou. */
export function ListSkeleton({ count = 3 }: { count?: number }) {
  return (
    <div className="flex flex-col gap-3" aria-busy="true" aria-label="Carregando">
      {Array.from({ length: count }, (_, index) => (
        <Skeleton key={index} className="h-20 w-full rounded-xl" />
      ))}
    </div>
  )
}
