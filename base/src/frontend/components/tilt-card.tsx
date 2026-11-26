import { useRef, type CSSProperties, type PointerEvent, type ReactNode } from "react"

import { cn } from "@/lib/utils"

interface Props {
  children: ReactNode
  className?: string
  /** O quanto inclina, em graus, nas bordas (padrão 7). */
  max?: number
}

/**
 * Inclina o conteúdo em 3D seguindo o mouse (perspectiva + rotateX/rotateY) e traz um brilho que acompanha o ponteiro.
 * No toque e com "reduzir movimento" ligado fica parado.
 */
export function TiltCard({ children, className, max = 7 }: Props) {
  const ref = useRef<HTMLDivElement>(null)

  function move(event: PointerEvent<HTMLDivElement>) {
    const element = ref.current

    if (!element || event.pointerType !== "mouse") return

    const box = element.getBoundingClientRect()
    const x = (event.clientX - box.left) / box.width
    const y = (event.clientY - box.top) / box.height

    element.style.setProperty("--rx", `${((0.5 - y) * 2 * max).toFixed(2)}deg`)
    element.style.setProperty("--ry", `${((x - 0.5) * 2 * max).toFixed(2)}deg`)
    element.style.setProperty("--gx", `${(x * 100).toFixed(0)}%`)
    element.style.setProperty("--gy", `${(y * 100).toFixed(0)}%`)
  }

  function leave() {
    const element = ref.current

    if (!element) return

    for (const name of ["--rx", "--ry"]) element.style.setProperty(name, "0deg")
  }

  const style = {
    transform: "perspective(900px) rotateX(var(--rx, 0deg)) rotateY(var(--ry, 0deg))",
    transformStyle: "preserve-3d",
  } satisfies CSSProperties

  return (
    <div
      ref={ref}
      onPointerMove={move}
      onPointerLeave={leave}
      style={style}
      className={cn(
        "group/tilt relative rounded-xl transition-transform duration-200 ease-out will-change-transform motion-reduce:!transform-none",
        className,
      )}
    >
      {children}
      {/* brilho de luz que segue o ponteiro */}
      <span
        aria-hidden="true"
        className="pointer-events-none absolute inset-0 rounded-[inherit] opacity-0 transition-opacity duration-200 group-hover/tilt:opacity-100 motion-reduce:hidden"
        style={{ background: "radial-gradient(14rem circle at var(--gx, 50%) var(--gy, 0%), color-mix(in oklch, var(--butter) 28%, transparent), transparent 70%)" }}
      />
    </div>
  )
}
