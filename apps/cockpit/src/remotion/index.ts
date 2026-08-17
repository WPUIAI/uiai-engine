// @ts-nocheck
import { registerRoot, Composition } from "remotion";
import { WorkRouterComposition } from "./WorkRouterComposition";

export const RemotionRoot = () => {
  return null as any;
};

export const compositions = [
  {
    id: "WorkRouter",
    component: WorkRouterComposition,
    durationInFrames: 180,
    fps: 30,
    width: 1920,
    height: 1080,
    defaultProps: { title: "Stop hunting for work. Let WorkRouter hunt for you.", personality: "Premium" as const },
  },
];

registerRoot(RemotionRoot);
