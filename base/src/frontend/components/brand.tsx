import mascot from "@/assets/mascote.webp"
import { cn } from "cn"

/** Marca do sistema: o mascote (vaca de óculos com o balde de leite) e o nome. */
export function Brand({ className, size = "md" }: { className?: string; size?: "sm" | "md" }) {
  return (
    <span className={cn("flex items-center gap-2.5", className)}>
      <img
        src={mascot}
        alt=""
        width={64}
        height={64}
        className={cn("shrink-0 rounded-full", size === "sm" ? "size-8" : "size-10")}
      />
      <span className="flex flex-col leading-none">
        <span className="font-heading text-base font-semibold tracking-tight">Análise Leite</span>
        <span className="text-[0.7rem] text-muted-foreground">Rastreio e qualidade do leite</span>
      </span>
    </span>
  )
}
