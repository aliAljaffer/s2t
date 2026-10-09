class S2t < Formula
  desc "Decode Kubernetes Secrets into readable key/value pairs"
  homepage "https://github.com/alialjaffer/s2t"
  url "https://github.com/alialjaffer/s2t/archive/refs/tags/v0.1.0.tar.gz"
  sha256 "a8d45857ea6e4c80185805dc8ce63f332018a2d2769af28c62bc67141db7ac6e"
  license "MIT"
  head "https://github.com/alialjaffer/s2t.git", branch: "main"

  depends_on "go" => :build

  def install
    # std_go_args already supplies -s -w and -trimpath
    ldflags = %W[-X main.version=#{version}]
    system "go", "build", *std_go_args(ldflags:), "."
  end

  test do
    assert_match "s2t version #{version}", shell_output("#{bin}/s2t --version")

    (testpath/"secret.yaml").write <<~YAML
      apiVersion: v1
      kind: Secret
      metadata:
        name: test
      data:
        password: c3VwZXItc2VjcmV0
    YAML
    assert_match "password: super-secret", shell_output("#{bin}/s2t -f #{testpath}/secret.yaml")
  end
end
