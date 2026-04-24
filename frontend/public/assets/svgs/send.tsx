import { SVGProps } from "react";

export const Send = (props: SVGProps<SVGSVGElement>) => (
  <svg
    width="100%"
    height="100%"
    preserveAspectRatio="xMidYMid meet"
    viewBox="0 0 24 24"
    {...props}
  >
    <g fill="none" strokeWidth="1.5">
      <g
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        clipPath="url(#send-clip)"
      >
        <path d="M22.152 3.553L11.178 21.004l-1.67-8.596L2 7.898l20.152-4.345ZM9.456 12.444l12.696-8.89" />
      </g>
      <defs>
        <clipPath id="send-clip">
          <path fill="#fff" d="M0 0h24v24H0z" />
        </clipPath>
      </defs>
    </g>
  </svg>
);
