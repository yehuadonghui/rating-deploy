import { Spin } from "antd"

export default function Loading() {
  return (
    <div style={{ display: "flex", justifyContent: "center", padding: 48 }}>
      <Spin size="large" />
    </div>
  )
}
