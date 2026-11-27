import type { CSSProperties } from "react"

import { cn } from "@/lib/utils"

/** Gotas (esferas de leite) que flutuam em volta, em posições e profundidades diferentes. */
const DROPS = [
  { top: "6%", left: "8%", size: 16, z: 70, delay: "0s", color: "var(--butter)" },
  { top: "18%", left: "88%", size: 11, z: -30, delay: "-1.2s", color: "var(--sky)" },
  { top: "78%", left: "4%", size: 12, z: 40, delay: "-2.4s", color: "var(--primary)" },
  { top: "84%", left: "86%", size: 18, z: 90, delay: "-3.4s", color: "var(--butter)" },
  { top: "46%", left: "-2%", size: 9, z: -50, delay: "-4.4s", color: "var(--sky)" },
]

/**
 * Cena 3D decorativa, só com CSS: anéis que giram em volta (como órbitas) e gotas de leite que flutuam em profundidade.
 * Vai atrás do conteúdo (ex.: o mascote). Não captura toque e some para leitores de tela.
 */
export function MilkOrbit({ className }: { className?: string }) {
  return (
    <div aria-hidden="true" className={cn("pointer-events-none relative [perspective:700px]", className)}>
      <div className="preserve-3d absolute inset-0">
        {[
          { inset: "-6%", tiltX: "64deg", tiltY: "0deg", duration: "16s", border: "color-mix(in oklch, var(--primary) 55%, transparent)" },
          { inset: "-16%", tiltX: "70deg", tiltY: "28deg", duration: "24s", border: "color-mix(in oklch, var(--butter) 80%, transparent)" },
          { inset: "-28%", tiltX: "58deg", tiltY: "-24deg", duration: "34s", border: "color-mix(in oklch, var(--sky) 70%, transparent)" },
        ].map((ring) => (
          <span
            key={ring.inset}
            className="orbit-ring absolute rounded-full border-2 border-dashed"
            style={
              {
                inset: ring.inset,
                borderColor: ring.border,
                "--tilt-x": ring.tiltX,
                "--tilt-y": ring.tiltY,
                transform: `rotateX(${ring.tiltX}) rotateY(${ring.tiltY})`,
                animation: `orbit-spin ${ring.duration} linear infinite`,
              } as CSSProperties
            }
          />
        ))}

        {DROPS.map((drop, index) => (
          <span
            key={index}
            className="orbit-drop absolute rounded-full"
            style={
              {
                top: drop.top,
                left: drop.left,
                width: drop.size,
                height: drop.size,
                "--z": `${drop.z}px`,
                background: `radial-gradient(circle at 30% 28%, white, ${drop.color} 62%)`,
                boxShadow: `0 6px 14px -4px color-mix(in oklch, ${drop.color} 70%, transparent)`,
                animation: `drop-float 7s ease-in-out ${drop.delay} infinite`,
              } as CSSProperties
            }
          />
        ))}
      </div>
    </div>
  )
}
