import { Sun, Moon } from "lucide-react";
import { cn } from "../lib/utils";
import { useTheme } from "../contexts/ThemeContext";

export function ThemeToggle() {
    const { theme, toggleTheme } = useTheme();
    const isDark = theme === "dark";

    return (
        <button
            type="button"
            onClick={toggleTheme}
            className={cn(
                "relative inline-flex h-8 w-14 items-center rounded-full border-4 transition-colors duration-300",
                isDark 
                    ? "border-indigo-400 bg-indigo-900" 
                    : "border-yellow-400 bg-yellow-200"
            )}
            aria-label={isDark ? "Switch to light mode" : "Switch to dark mode"}
        >
            {/* Sliding thumb */}
            <span
                className={cn(
                    "absolute flex h-5 w-5 items-center justify-center rounded-full shadow-md transition-all duration-300",
                    isDark 
                        ? "translate-x-6 bg-indigo-200" 
                        : "translate-x-0.5 bg-yellow-100"
                )}
            >
                {isDark ? (
                    <Moon className="h-3 w-3 text-indigo-700" />
                ) : (
                    <Sun className="h-3 w-3 text-yellow-600" />
                )}
            </span>
            
            {/* Icons on sides */}
            <span className="absolute left-1.5 flex items-center justify-center">
                <Sun className={cn(
                    "h-3 w-3 transition-opacity duration-300",
                    isDark ? "opacity-30" : "opacity-100 text-yellow-600"
                )} />
            </span>
            <span className="absolute right-1.5 flex items-center justify-center">
                <Moon className={cn(
                    "h-3 w-3 transition-opacity duration-300",
                    isDark ? "opacity-100 text-indigo-300" : "opacity-30"
                )} />
            </span>
        </button>
    );
}
