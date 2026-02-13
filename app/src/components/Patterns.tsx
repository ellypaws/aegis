import React from "react";

export const Patterns = {
    Polka({ color = "rgba(255,0,0,0.10)" }: { color?: string }) {
        return (
            <div
                aria-hidden
                className="pointer-events-none absolute inset-0 opacity-100"
                style={{
                    backgroundImage: `radial-gradient(${color} 2px, transparent 2px)`,
                    backgroundSize: "18px 18px",
                    backgroundPosition: "0 0",
                }}
            />
        );
    },
};
