import { useState, forwardRef } from "react";
import { cn } from "../lib/utils";

interface ImageWithSpinnerProps {
    src: string;
    srcSet?: string;
    sizes?: string;
    alt: string;
    className?: string;
    style?: React.CSSProperties;
    draggable?: boolean;
}

export const ImageWithSpinner = forwardRef<HTMLImageElement, ImageWithSpinnerProps>(({
    src,
    srcSet,
    sizes,
    alt,
    className,
    style,
    draggable = false,
}, ref) => {
    const [isLoading, setIsLoading] = useState(true);

    return (
        <>
            {isLoading && (
                <div className="absolute inset-0 flex items-center justify-center bg-zinc-100 dark:bg-zinc-800 z-10 pointer-events-none">
                    <div className="h-5 w-5 animate-spin rounded-full border-2 border-zinc-300 border-t-zinc-600 dark:border-zinc-600 dark:border-t-zinc-300" />
                </div>
            )}
            <img
                ref={ref}
                src={src}
                srcSet={srcSet}
                sizes={sizes}
                alt={alt}
                className={cn(className, isLoading ? "opacity-0" : "opacity-100")}
                style={style}
                draggable={draggable}
                loading="lazy"
                onLoad={() => setIsLoading(false)}
            />
        </>
    );
});

ImageWithSpinner.displayName = "ImageWithSpinner";
