// �p�����[�^�錾


float4 MaterialDiffuse   : DIFFUSE  < string Object = "Geometry"; >;
// ���@�ϊ��s��
float4x4 matWVP      : WORLDVIEWPROJECTION;
float4x4 matW	     : WORLD;

float3   CameraPosition    : POSITION  < string Object = "Camera"; >;

// MMD�{����sampler���㏑�����Ȃ����߂̋L�q�ł��B�폜�s�B
sampler MMDSamp0 : register(s0);
sampler MMDSamp1 : register(s1);
sampler MMDSamp2 : register(s2);

// �I�u�W�F�N�g�̃e�N�X�`��
texture ObjectTexture: MATERIALTEXTURE;
sampler ObjTexSampler = sampler_state {
    texture = <ObjectTexture>;
    MINFILTER = LINEAR;
    MAGFILTER = LINEAR;
};


///////////////////////////////////////////////////////////////////////////////////////////////
// �I�u�W�F�N�g�`��i�Z���t�V���h�EOFF�j

struct VS_OUTPUT
{
    float4 Pos        : POSITION;    // �ˉe�ϊ����W
    float3 Normal  	  : TEXCOORD0;
    float2 ObjTex	  : TEXCOORD1;
};

// ���_�V�F�[�_
VS_OUTPUT Basic_VS(float4 Pos : POSITION, float3 Normal : NORMAL,float2 Tex: TEXCOORD0)
{
    VS_OUTPUT Out = (VS_OUTPUT)0;

    Out.Pos = mul( Pos, matWVP );
    Out.Normal = mul(float4(normalize(Normal),1),matW).xyz;
    Out.ObjTex = Tex;
    return Out;
}

// �s�N�Z���V�F�[�_
float4 Basic_PS( VS_OUTPUT IN, uniform bool useTex ) : COLOR
{
	float alpha = MaterialDiffuse.a;

	if(useTex)
	{
		alpha = tex2D(ObjTexSampler,IN.ObjTex).a;
	}
	return float4(IN.Normal*0.5+0.5,alpha > 0.9);
}

// �I�u�W�F�N�g�`��p�e�N�j�b�N
technique MainTec_1 < string MMDPass = "object"; bool UseTexture = false;> {
    pass DrawObject
    {
        VertexShader = compile vs_2_0 Basic_VS();
        PixelShader  = compile ps_2_0 Basic_PS(false);
    }
}

// �I�u�W�F�N�g�`��p�e�N�j�b�N
technique MainTecBS_1  < string MMDPass = "object_ss"; bool UseTexture = false;> {
    pass DrawObject {
        AlphaBlendEnable = FALSE;
        VertexShader = compile vs_2_0 Basic_VS();
        PixelShader  = compile ps_2_0 Basic_PS(false);
    }
}
// �I�u�W�F�N�g�`��p�e�N�j�b�N
technique MainTec_2 < string MMDPass = "object"; bool UseTexture = true;> {
    pass DrawObject
    {
        VertexShader = compile vs_2_0 Basic_VS();
        PixelShader  = compile ps_2_0 Basic_PS(true);
    }
}

// �I�u�W�F�N�g�`��p�e�N�j�b�N
technique MainTecBS_2  < string MMDPass = "object_ss"; bool UseTexture = true;> {
    pass DrawObject {
        AlphaBlendEnable = FALSE;
        VertexShader = compile vs_2_0 Basic_VS();
        PixelShader  = compile ps_2_0 Basic_PS(true);
    }
}
technique EdgeTec < string MMDPass = "edge"; > {

}
technique ShadowTech < string MMDPass = "shadow";  > {
    
}

///////////////////////////////////////////////////////////////////////////////////////////////
