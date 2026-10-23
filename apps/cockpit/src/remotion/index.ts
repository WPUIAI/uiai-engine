// @ts-nocheck
import React from "react";
import { registerRoot, Composition } from "remotion";
import { WorkRouterComposition } from "./WorkRouterComposition";

export const RemotionRoot = () => {
  return React.createElement(React.Fragment, null,
    React.createElement(Composition, {
      id: "WorkRouter",
      component: WorkRouterComposition,
      durationInFrames: 180,
      fps: 30,
      width: 1920,
      height: 1080,
      defaultProps: { title: "Stop hunting for work. Let WorkRouter hunt for you.", personality: "Premium", shaderEnabled: true, particlesEnabled: true, lottieEnabled: false, lowerThirdEnabled: true }
    })
  );
};
registerRoot(RemotionRoot);
