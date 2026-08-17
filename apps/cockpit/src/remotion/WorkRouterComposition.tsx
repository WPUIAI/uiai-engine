// @ts-nocheck
import { AbsoluteFill, Sequence, useCurrentFrame, useVideoConfig, interpolate, Easing } from "remotion";

export const WorkRouterComposition = ({ title = "Stop hunting for work. Let WorkRouter hunt for you.", personality = "Premium" }) => {
  const frame = useCurrentFrame();
  const cfg = {
    Premium: { dur: 15, ease: Easing.bezier(0.4, 0, 0.2, 1), stagger: 2 },
    Playful: { dur: 8, ease: Easing.bezier(0.175, 0.885, 0.32, 1.275), stagger: 2 },
    Corporate: { dur: 9, ease: Easing.bezier(0.2, 0, 0, 1), stagger: 1.5 },
    Energetic: { dur: 6, ease: Easing.out(Easing.exp), stagger: 1 },
  }[personality];
  const p1 = interpolate(frame, [0, 12], [0, 1], { extrapolateLeft: "clamp", extrapolateRight: "clamp", easing: cfg.ease });
  const p2 = interpolate(frame, [36, 48], [0, 1], { extrapolateLeft: "clamp", extrapolateRight: "clamp", easing: cfg.ease });
  const bloom = 0.85 + Math.sin(frame * 0.05) * 0.07;
  const words = title.split(" ");
  return (
    <AbsoluteFill style={{ background: "#0a0a14", color: "#fff", overflow: "hidden" }}>
      <Sequence from={0} durationInFrames={45}>
        <div style={{ position: "absolute", inset: 0, opacity: p1, transform: `scale(${0.96 + p1 * 0.04})` }}>
          <div style={{ width: "100%", height: "100%", background: "linear-gradient(135deg,#1a1a2e 0%,#16213e 50%,#0f3460 100%)", display: "flex", alignItems: "center", justifyContent: "center", fontSize: 28, fontWeight: 800 }}>workrouter.app — real capture</div>
        </div>
      </Sequence>
      <Sequence from={36} durationInFrames={58}>
        <div style={{ position: "absolute", inset: 0, opacity: p2, transform: `perspective(900px) rotateY(${0.22 + Math.sin(frame*0.04)*0.12}rad) rotateX(${-0.35 + Math.cos(frame*0.03)*0.06}rad)` }}>
          <div style={{ width: 860, height: 520, margin: "auto", marginTop: 60, background: "linear-gradient(135deg,#3355ff 0%,#6a7bff 100%)", borderRadius: 16, display: "grid", gridTemplateColumns: "repeat(3,1fr)", gap: 12, padding: 16 }}>
            {[1,2,3,4,5,6].map((i) => (<div key={i} style={{ background: "#fff", borderRadius: 12, padding: 12, opacity: p2 }}><div style={{ height: 8, background: "#e5e7eb", borderRadius: 4, marginBottom: 8 }} /><div style={{ height: 40, background: "#f3f4f6", borderRadius: 8 }} /></div>))}
          </div>
        </div>
      </Sequence>
      <Sequence from={84} durationInFrames={60}>
        <div style={{ position: "absolute", left: 0, right: 0, top: "42%", textAlign: "center", padding: "0 40px" }}>
          <h1 style={{ fontSize: 44, fontWeight: 900, lineHeight: 1.1 }}>
            {words.map((w, i) => {
              const d = i * cfg.stagger;
              const wp = interpolate(frame, [84 + d, 84 + d + cfg.dur], [0, 1], { extrapolateLeft: "clamp", extrapolateRight: "clamp", easing: cfg.ease });
              return (<span key={i} style={{ display: "inline-block", marginRight: 10, opacity: wp, transform: `translateY(${40 - wp*40}px)`, filter: `blur(${10 - wp*10}px)` }}>{w}</span>);
            })}
          </h1>
        </div>
      </Sequence>
      <Sequence from={135} durationInFrames={45}>
        <div style={{ position: "absolute", bottom: 24, left: "50%", transform: "translateX(-50%)", background: "rgba(0,0,0,0.8)", color: "#fff", padding: "8px 14px", borderRadius: 8, fontSize: 11, opacity: interpolate(frame, [135, 142], [0, 1], { extrapolateLeft: "clamp", extrapolateRight: "clamp" }) }}>WEBVTT · Let WorkRouter hunt for you · bloom {bloom.toFixed(2)}</div>
      </Sequence>
    </AbsoluteFill>
  );
};
