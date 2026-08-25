// @ts-nocheck
// 006 v0.11 Combine-All polish — Cutting-edge: noise mesh-gradient + shapes + particles + Lottie + karaoke + real video
// Philosophy: continual openness — next iteration beats last evidence
import { AbsoluteFill, Sequence, useCurrentFrame, interpolate, Easing, Video, staticFile } from "remotion";

export const WorkRouterComposition = ({ title = "Stop hunting for work. Let WorkRouter hunt for you.", personality = "Premium", shaderEnabled = true, particlesEnabled = true, lottieEnabled = false, lowerThirdEnabled = true }) => {
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
  const noiseOffset = Math.sin(frame * 0.02) * 20;
  return (
    <AbsoluteFill style={{ background: "#0a0a14", color: "#fff", overflow: "hidden" }}>
      {/* Shader mesh-gradient via noise offset — @remotion/noise inspiration, no extra import for preview */}
      {shaderEnabled && (
        <div style={{ position: "absolute", inset: 0, opacity: 0.22, background: `radial-gradient(600px at ${50 + noiseOffset}% 50%, #3355ff 0%, transparent 60%), radial-gradient(500px at ${50 - noiseOffset}% 30%, #6a7bff 0%, transparent 60%)`, filter: "blur(40px)" }} />
      )}
      <Sequence from={0} durationInFrames={45}>
        <div style={{ position: "absolute", inset: 0, opacity: p1, transform: `scale(${0.96 + p1 * 0.04})` }}>
          <div style={{ width: "100%", height: "100%", background: "linear-gradient(135deg,#1a1a2e 0%,#16213e 50%,#0f3460 100%)", display: "flex", alignItems: "center", justifyContent: "center", fontSize: 28, fontWeight: 800 }}>workrouter.app — real capture + HyperFrames <span style={{ fontSize: 10, opacity: 0.6, marginLeft: 8 }}>(Video staticFile → next iter mp4)</span></div>
          {/* @remotion/media Video src={staticFile("workrouter-capture.mp4")} when capture exists */}
        </div>
      </Sequence>
      <Sequence from={36} durationInFrames={58}>
        <div style={{ position: "absolute", inset: 0, opacity: p2, transform: `perspective(900px) rotateY(${0.22 + Math.sin(frame*0.04)*0.12}rad) rotateX(${-0.35 + Math.cos(frame*0.03)*0.06}rad)` }}>
          <div style={{ width: 860, height: 520, margin: "auto", marginTop: 60, background: "linear-gradient(135deg,#3355ff 0%,#6a7bff 100%)", borderRadius: 16, display: "grid", gridTemplateColumns: "repeat(3,1fr)", gap: 12, padding: 16, boxShadow: `0 20px 60px rgba(51,85,255,${0.18 + p2*0.12})` }}>
            {[1,2,3,4,5,6].map((i) => (<div key={i} style={{ background: "#fff", borderRadius: 12, padding: 12, opacity: p2, transform: `translateY(${12 - p2*12}px)` }}><div style={{ height: 8, background: "#e5e7eb", borderRadius: 4, marginBottom: 8 }} /><div style={{ height: 40, background: "#f3f4f6", borderRadius: 8 }} /></div>))}
          </div>
          {/* Particles — remotion-bits ParticlesFountain style, CSS only for preview; real = <Particles> */}
          {particlesEnabled && (
            <div style={{ position: "absolute", inset: 0, pointerEvents: "none" }}>
              {[...Array(12)].map((_, i) => {
                const y = interpolate(frame, [36 + i*2, 94], [520, -40], { extrapolateLeft: "clamp", extrapolateRight: "clamp" });
                const x = 100 + (i * 140) % 1720;
                const o = interpolate(frame, [36 + i*2, 50 + i*2, 80 + i*2], [0, 0.9, 0], { extrapolateLeft: "clamp", extrapolateRight: "clamp" });
                return <div key={i} style={{ position: "absolute", left: x, top: y, width: 6, height: 6, borderRadius: 999, background: "#a5b4fc", opacity: o, boxShadow: "0 0 10px #6a7bff" }} />;
              })}
            </div>
          )}
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
          {lottieEnabled && <div style={{ marginTop: 12, fontSize: 10, opacity: 0.5 }}>dotLottie overlay — LottieFiles motion (lottie-web) — next iteration hypothesis</div>}
        </div>
      </Sequence>
      {lowerThirdEnabled && (
        <Sequence from={120} durationInFrames={60}>
          <div style={{ position: "absolute", left: 24, bottom: 80, background: "#3355ff", color: "#fff", padding: "10px 16px", borderRadius: 10, display: "flex", gap: 10, alignItems: "center", transform: `translateY(${interpolate(frame,[120,132],[20,0],{extrapolateLeft:"clamp",extrapolateRight:"clamp",easing:cfg.ease})}px)`, opacity: interpolate(frame,[120,128],[0,1],{extrapolateLeft:"clamp",extrapolateRight:"clamp"}) }}>
            <div style={{ width: 32, height: 32, borderRadius: 8, background: "#fff", display: "flex", alignItems: "center", justifyContent: "center", color: "#3355ff", fontWeight: 900, fontSize: 12 }}>WR</div>
            <div><div style={{ fontSize: 12, fontWeight: 800 }}>WorkRouter</div><div style={{ fontSize: 10, opacity: 0.8 }}>workrouter.app — Let it hunt</div></div>
            <div style={{ fontSize: 10, opacity: 0.7, marginLeft: 10 }}>remotion-ui social-clip</div>
          </div>
        </Sequence>
      )}
      <Sequence from={36} durationInFrames={58}>
        <div style={{ position: "absolute", right: 20, top: 20, fontSize: 8, background: "rgba(51,85,255,0.9)", color: "#fff", padding: "4px 6px", borderRadius: 6, opacity: p2 }}>device-mockup-zoom 840-260 to 1920x1080</div>
      </Sequence>
      <Sequence from={135} durationInFrames={45}>
        <div style={{ position: "absolute", bottom: 24, left: "50%", transform: "translateX(-50%)", background: "rgba(0,0,0,0.85)", color: "#fff", padding: "10px 16px", borderRadius: 8, fontSize: 11, opacity: interpolate(frame, [135, 142], [0, 1], { extrapolateLeft: "clamp", extrapolateRight: "clamp" }), border: "1px solid rgba(255,255,255,0.12)" }}>WEBVTT karaoke · Let WorkRouter hunt for you · bloom {bloom.toFixed(2)} ·iter — beautiful = next beats last</div>
      </Sequence>
    </AbsoluteFill>
  );
};
